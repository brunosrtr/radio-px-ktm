import 'dart:math' as math;
import 'dart:typed_data';

import 'package:flutter/services.dart';
import 'package:flutter_sound/flutter_sound.dart';

/// O "bip" de cada leitura enviada (FR-007): vibração curta e um tom de
/// 0,15 s gerado em memória — nada é gravado em disco. O usuário pode
/// desligar o som e a vibração sem desligar o ponto de transmissão.
class Bip {
  FlutterSoundPlayer? _player;
  Uint8List? _tom;

  Future<void> tocar({required bool som, required bool vibracao}) async {
    if (vibracao) {
      await HapticFeedback.heavyImpact();
    }
    if (!som) return;
    try {
      final player = _player ??= FlutterSoundPlayer();
      if (!player.isOpen()) await player.openPlayer();
      await player.startPlayer(fromDataBuffer: _tom ??= gerarTomWav(), codec: Codec.pcm16WAV);
    } catch (_) {
      // Sem som não é motivo para perder a leitura: a vibração já avisou.
    }
  }

  Future<void> fechar() async {
    await _player?.closePlayer();
    _player = null;
  }
}

/// Gera um WAV PCM 16 bits mono com dois tons curtos (estilo "bip-bip").
Uint8List gerarTomWav({int taxa = 22050}) {
  const trechos = [(1760.0, 0.07), (0.0, 0.03), (2349.0, 0.07)];
  final amostras = <int>[];
  for (final (frequencia, segundos) in trechos) {
    final n = (taxa * segundos).round();
    for (var i = 0; i < n; i++) {
      final envelope = math.min(1.0, math.min(i, n - i) / (taxa * 0.005));
      final valor = frequencia == 0 ? 0.0 : math.sin(2 * math.pi * frequencia * i / taxa) * 0.45 * envelope;
      amostras.add((valor * 32767).round());
    }
  }
  final dados = ByteData(44 + amostras.length * 2);
  void texto(int pos, String s) {
    for (var i = 0; i < s.length; i++) {
      dados.setUint8(pos + i, s.codeUnitAt(i));
    }
  }

  texto(0, 'RIFF');
  dados.setUint32(4, 36 + amostras.length * 2, Endian.little);
  texto(8, 'WAVE');
  texto(12, 'fmt ');
  dados.setUint32(16, 16, Endian.little);
  dados.setUint16(20, 1, Endian.little); // PCM
  dados.setUint16(22, 1, Endian.little); // mono
  dados.setUint32(24, taxa, Endian.little);
  dados.setUint32(28, taxa * 2, Endian.little);
  dados.setUint16(32, 2, Endian.little);
  dados.setUint16(34, 16, Endian.little);
  texto(36, 'data');
  dados.setUint32(40, amostras.length * 2, Endian.little);
  for (var i = 0; i < amostras.length; i++) {
    dados.setInt16(44 + i * 2, amostras[i], Endian.little);
  }
  return dados.buffer.asUint8List();
}
