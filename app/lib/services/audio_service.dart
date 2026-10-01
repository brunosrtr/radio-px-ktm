import 'dart:async';
import 'dart:typed_data';

import 'package:flutter_sound/flutter_sound.dart';

/// Grava e reproduz voz em streaming — cada chunk Opus trafega direto entre
/// o microfone/alto-falante e o `canal_service.dart`, sem tocar disco em
/// nenhum momento (Princípio I da constituição, RF-08/RNF-05/RNF-06).
///
/// O player e o gravador abrem separados de propósito: abrir o gravador pede
/// permissão de microfone ao navegador, que pode ficar pendente esperando o
/// usuário decidir. Quem só quer ouvir o canal não deveria travar nisso —
/// por isso o gravador só abre sob demanda, na primeira vez que o usuário
/// aperta o botão de falar (`iniciarGravacao`).
class AudioService {
  static const _codec = Codec.opusOGG;
  static const _sampleRate = 16000;
  static const _bitRateBps = 24000; // RNF-14: no máximo 24 kbps

  final FlutterSoundRecorder _recorder = FlutterSoundRecorder();
  final FlutterSoundPlayer _player = FlutterSoundPlayer();

  StreamController<Uint8List>? _chunksGravados;
  bool _playerAberto = false;
  bool _recorderAberto = false;

  Future<void> abrirParaOuvir() async {
    if (_playerAberto) return;
    await _player.openPlayer();
    _playerAberto = true;
  }

  /// Destrava o áudio no navegador: o `AudioContext` criado por
  /// `openPlayer()` nasce suspenso pela política de autoplay do Chrome e só
  /// toca som de verdade depois de retomado dentro de uma interação real do
  /// usuário (toque/clique) — sem isso, os eventos de fala chegam
  /// normalmente e nada é ouvido. Seguro chamar mais de uma vez.
  Future<void> destravarAudioNoToque() async {
    if (!_playerAberto) return;
    try {
      await _player.resumePlayer();
    } catch (_) {
      // resumePlayer pode falhar se o contexto já estiver rodando — não é
      // um erro que precise interromper nada.
    }
  }

  Future<void> _abrirGravadorSeNecessario() async {
    if (_recorderAberto) return;
    await _recorder.openRecorder();
    _recorderAberto = true;
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

  /// Abre o gravador (pedindo permissão de microfone, se ainda não concedida)
  /// e inicia a gravação, entregando cada chunk assim que é gerado, sem
  /// esperar o fim da fala (research.md §7) — quem chama repassa cada evento
  /// do stream ao `canal_service.dart` como frame binário.
  Future<Stream<Uint8List>> iniciarGravacao() async {
    await _abrirGravadorSeNecessario();

    final controlador = StreamController<Uint8List>();
    _chunksGravados = controlador;

    try {
      await _recorder.startRecorder(
        codec: _codec,
        toStream: controlador.sink,
        sampleRate: _sampleRate,
        numChannels: 1,
        bitRate: _bitRateBps,
      );
    } catch (_) {
      // Uma falha antes de devolver o stream não pode deixar close()
      // aguardando indefinidamente um ouvinte que nunca será registrado.
      await controlador.stream.listen((_) {}).cancel();
      await controlador.close();
      _chunksGravados = null;
      rethrow;
    }

    return controlador.stream;
  }

  Future<void> pararGravacao() async {
    if (_recorder.isRecording) {
      await _recorder.stopRecorder();
    }
    await _chunksGravados?.close();
    _chunksGravados = null;
  }

  Future<void> iniciarReproducao() {
    return _player.startPlayerFromStream(
      codec: _codec,
      interleaved: true,
      numChannels: 1,
      sampleRate: _sampleRate,
      bufferSize: 4096,
    );
  }

  /// Encaminha um chunk recebido do canal para reprodução imediata — nunca
  /// gravado, apenas repassado ao alto-falante.
  Future<void> reproduzirChunk(Uint8List chunk) => _player.feedUint8FromStream(chunk);

  Future<void> pararReproducao() async {
    if (_player.isPlaying) {
      await _player.stopPlayer();
    }
  }
}
