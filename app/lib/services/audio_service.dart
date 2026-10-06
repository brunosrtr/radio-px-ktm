import 'dart:async';
import 'dart:collection';
import 'dart:typed_data';

import 'package:flutter/foundation.dart' show kIsWeb;
import 'package:flutter_sound/flutter_sound.dart';
import 'package:opus_dart/opus_dart.dart';
import 'package:opus_flutter_new/opus_flutter_new.dart' as opus_flutter;
import 'package:permission_handler/permission_handler.dart';

import 'codec_voz.dart';

class FalhaAudio implements Exception {
  const FalhaAudio(this.mensagem);
  final String mensagem;
  @override
  String toString() => mensagem;
}

/// Captura/reprodução PCM em memória; somente Opus trafega pela rede.
/// Nenhum caminho de arquivo é passado ao gravador ou ao player.
class AudioService {
  static Future<void>? _opusPronto;
  final FlutterSoundRecorder _recorder = FlutterSoundRecorder();
  final FlutterSoundPlayer _player = FlutterSoundPlayer();
  void Function(Object)? aoFalharReproducao;
  StreamController<Uint8List>? _pcmGravado;
  StreamController<Uint8List>? _opusGravado;
  StreamSubscription<Uint8List>? _captura;
  CodificadorVoz? _encoder;
  DecodificadorVoz? _decoder;
  bool _playerAberto = false;
  bool _recorderAberto = false;
  bool _reproduzindo = false;
  int _geracao = 0;
  final _fila = Queue<Uint8List>();
  Future<void>? _alimentacao;

  Future<void> abrirParaOuvir() async {
    if (_playerAberto) return;
    try {
      await (_opusPronto ??= _carregarOpus());
      await _player.openPlayer();
      _playerAberto = true;
    } catch (_) {
      throw const FalhaAudio(
        'Não foi possível preparar o áudio. Saia do canal e tente novamente.',
      );
    }
  }

  static Future<void> _carregarOpus() async {
    try {
      initOpus(await opus_flutter.load());
    } catch (_) {
      _opusPronto = null;
      rethrow;
    }
  }

  Future<void> destravarAudioNoToque() async {
    if (!kIsWeb || !_playerAberto) return;
    await _player.resumePlayer();
  }

  Future<void> _abrirGravadorSeNecessario() async {
    if (_recorderAberto) return;
    if (!kIsWeb && !await Permission.microphone.request().isGranted) {
      throw const FalhaAudio('Permita o acesso ao microfone para falar.');
    }
    await _recorder.openRecorder();
    _recorderAberto = true;
  }

  Future<Stream<Uint8List>> iniciarGravacao() async {
    await _abrirGravadorSeNecessario();
    _encoder = CodificadorVoz();
    final pcm = _pcmGravado = StreamController<Uint8List>();
    final opus = _opusGravado = StreamController<Uint8List>();
    _captura = pcm.stream.listen((chunk) {
      try {
        for (final pacote in _encoder!.converter(chunk)) {
          opus.add(pacote);
        }
      } catch (erro, stack) {
        opus.addError(erro, stack);
      }
    }, onError: opus.addError);
    try {
      await _recorder.startRecorder(
        codec: Codec.pcm16,
        toStream: pcm.sink,
        sampleRate: CodificadorVoz.sampleRate,
        numChannels: 1,
        bufferSize: 2048,
      );
      return opus.stream;
    } catch (_) {
      // O PTT ainda não recebeu o stream: drenar antes de fechar.
      final descarte = opus.stream.listen((_) {}, onError: (Object _) {});
      await pararGravacao();
      await descarte.cancel();
      throw const FalhaAudio(
        'Não foi possível iniciar o microfone. Saia do canal e tente novamente.',
      );
    }
  }

  Future<void> pararGravacao() async {
    try {
      if (_recorder.isRecording) await _recorder.stopRecorder();
    } finally {
      await _pcmGravado?.close();
      await _captura?.cancel();
      _captura = null;
      _pcmGravado = null;
      _encoder?.fechar();
      _encoder = null;
      await _opusGravado?.close();
      _opusGravado = null;
    }
  }

  Future<void> iniciarReproducao() async {
    try {
      await _player.startPlayerFromStream(
        codec: Codec.pcm16,
        interleaved: true,
        numChannels: 1,
        sampleRate: CodificadorVoz.sampleRate,
        bufferSize: 4096,
      );
      _reproduzindo = true;
    } catch (_) {
      throw const FalhaAudio(
        'Não foi possível iniciar a saída de áudio. Saia do canal e tente novamente.',
      );
    }
  }

  /// Reinicia o estado Opus para cada fala/remetente.
  void iniciarRecepcao() {
    encerrarRecepcao();
    _decoder = DecodificadorVoz();
  }

  void encerrarRecepcao() {
    _decoder?.fechar();
    _decoder = null;
  }

  void reproduzirChunk(Uint8List chunk) {
    if (!_reproduzindo || _decoder == null) return;
    try {
      // No máximo as 10 falas de 90s previstas no protocolo, somente em RAM.
      if (_fila.length >= 45000) throw StateError('Fila de áudio excedida.');
      _fila.add(_decoder!.converter(chunk));
      _alimentacao ??= _alimentar(_geracao)
          .whenComplete(() => _alimentacao = null);
    } catch (erro) {
      _falharReproducao(erro);
    }
  }

  Future<void> _alimentar(int geracao) async {
    try {
      while (_fila.isNotEmpty && _reproduzindo && geracao == _geracao) {
        final pcm = _fila.removeFirst();
        var offset = 0;
        while (offset < pcm.length && _reproduzindo && geracao == _geracao) {
          final usados = await _player
              .feedUint8FromStream(Uint8List.sublistView(pcm, offset))
              .timeout(const Duration(seconds: 5));
          if (!_reproduzindo || geracao != _geracao) return;
          if (usados <= 0 || usados > pcm.length - offset) {
            throw StateError('O player não aceitou o áudio.');
          }
          offset += usados;
        }
      }
    } catch (erro) {
      if (_reproduzindo && geracao == _geracao) _falharReproducao(erro);
    }
  }

  void _falharReproducao(Object erro) {
    if (!_reproduzindo) return;
    unawaited(pararReproducao());
    aoFalharReproducao?.call(
      const FalhaAudio(
        'Não foi possível reproduzir a voz. Saia do canal e entre novamente.',
      ),
    );
  }

  Future<void> pararReproducao() async {
    _reproduzindo = false;
    _geracao++;
    _fila.clear();
    encerrarRecepcao();
    if (_player.isPlaying) await _player.stopPlayer();
    await _alimentacao;
  }

  Future<void> fechar() async {
    await pararGravacao();
    await pararReproducao();
    if (_recorderAberto) {
      await _recorder.closeRecorder();
      _recorderAberto = false;
    }
    if (_playerAberto) {
      await _player.closePlayer();
      _playerAberto = false;
    }
  }
}
