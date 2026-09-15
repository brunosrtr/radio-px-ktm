import 'dart:async';
import 'dart:typed_data';

import 'package:flutter_sound/flutter_sound.dart';

/// Grava e reproduz voz em streaming — cada chunk Opus trafega direto entre
/// o microfone/alto-falante e o `canal_service.dart`, sem tocar disco em
/// nenhum momento (Princípio I da constituição, RF-08/RNF-05/RNF-06).
class AudioService {
  static const _codec = Codec.opusOGG;
  static const _sampleRate = 16000;
  static const _bitRateBps = 24000; // RNF-14: no máximo 24 kbps

  final FlutterSoundRecorder _recorder = FlutterSoundRecorder();
  final FlutterSoundPlayer _player = FlutterSoundPlayer();

  StreamController<Uint8List>? _chunksGravados;
  bool _aberto = false;

  Future<void> abrir() async {
    if (_aberto) return;
    await _recorder.openRecorder();
    await _player.openPlayer();
    _aberto = true;
  }

  Future<void> fechar() async {
    if (!_aberto) return;
    await pararGravacao();
    await pararReproducao();
    await _recorder.closeRecorder();
    await _player.closePlayer();
    _aberto = false;
  }

  /// Inicia a gravação e entrega cada chunk assim que é gerado, sem esperar
  /// o fim da fala (research.md §7) — quem chama repassa cada evento do
  /// stream ao `canal_service.dart` como frame binário.
  Future<Stream<Uint8List>> iniciarGravacao() async {
    final controlador = StreamController<Uint8List>();
    _chunksGravados = controlador;

    await _recorder.startRecorder(
      codec: _codec,
      toStream: controlador.sink,
      sampleRate: _sampleRate,
      numChannels: 1,
      bitRate: _bitRateBps,
    );

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
