import 'dart:typed_data';

import 'package:opus_dart/opus_dart.dart';

/// PCM16 mono a 16 kHz local; um pacote Opus de 20 ms por frame WebSocket.
/// O teto de 60 bytes por pacote limita o áudio comprimido a 24 kbit/s.
class CodificadorVoz {
  static const sampleRate = 16000;
  static const amostrasPorPacote = 320;
  static const bytesPorFrame = amostrasPorPacote * 2;
  static const bytesMaximosOpus = 60;
  final _encoder = SimpleOpusEncoder(
    sampleRate: sampleRate,
    channels: 1,
    application: Application.voip,
  );
  final _pendente = Uint8List(bytesPorFrame);
  int _ocupados = 0;

  /// Aceita inclusive chunks que dividem uma amostra entre dois callbacks.
  List<Uint8List> converter(Uint8List pcm) {
    final pacotes = <Uint8List>[];
    var offset = 0;
    while (offset < pcm.length) {
      final quantidade = (pcm.length - offset).clamp(
        0,
        bytesPorFrame - _ocupados,
      );
      _pendente.setRange(_ocupados, _ocupados + quantidade, pcm, offset);
      offset += quantidade;
      _ocupados += quantidade;
      if (_ocupados == bytesPorFrame) {
        final dados = ByteData.sublistView(_pendente);
        final amostras = Int16List(amostrasPorPacote);
        for (var i = 0; i < amostras.length; i++) {
          amostras[i] = dados.getInt16(i * 2, Endian.little);
        }
        pacotes.add(
          _encoder.encode(
            input: amostras,
            maxOutputSizeBytes: bytesMaximosOpus,
          ),
        );
        _ocupados = 0;
      }
    }
    return pacotes;
  }

  void fechar() {
    _pendente.fillRange(0, _pendente.length, 0);
    _ocupados = 0;
    _encoder.destroy();
  }
}

class DecodificadorVoz {
  final _decoder = SimpleOpusDecoder(
    sampleRate: CodificadorVoz.sampleRate,
    channels: 1,
  );
  Uint8List converter(Uint8List opus) {
    if (opus.isEmpty || opus.length > CodificadorVoz.bytesMaximosOpus) {
      throw const FormatException('Pacote de voz inválido.');
    }
    final amostras = _decoder.decode(input: opus);
    if (amostras.length != CodificadorVoz.amostrasPorPacote) {
      throw const FormatException('Duração de pacote de voz inválida.');
    }
    final pcm = ByteData(amostras.length * 2);
    for (var i = 0; i < amostras.length; i++) {
      pcm.setInt16(i * 2, amostras[i], Endian.little);
    }
    return pcm.buffer.asUint8List();
  }

  void fechar() => _decoder.destroy();
}
