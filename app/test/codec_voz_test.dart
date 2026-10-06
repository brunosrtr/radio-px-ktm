import 'dart:ffi';
import 'dart:math';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:opus_dart/opus_dart.dart';
import 'package:radio_px_app/services/codec_voz.dart';

Uint8List tom(int frames) {
  final dados = ByteData(frames * CodificadorVoz.bytesPorFrame);
  for (var i = 0; i < dados.lengthInBytes ~/ 2; i++) {
    dados.setInt16(
      i * 2,
      (sin(2 * pi * 440 * i / 16000) * 8000).round(),
      Endian.little,
    );
  }
  return dados.buffer.asUint8List();
}

void main() {
  setUpAll(() {
    // A API do pacote alterna o tipo da biblioteca entre FFI nativa e web.
    final dynamic biblioteca = DynamicLibrary.open('libopus.so.0');
    initOpus(biblioteca);
  });

  test('Opus real preserva duração e sinal dentro de 24 kbit/s', () {
    final encoder = CodificadorVoz();
    final decoder = DecodificadorVoz();
    addTearDown(encoder.fechar);
    addTearDown(decoder.fechar);
    final pacotes = encoder.converter(tom(50));
    expect(pacotes, hasLength(50));
    var energia = 0;
    for (final pacote in pacotes) {
      expect(pacote.length, inInclusiveRange(1, 60));
      final pcm = decoder.converter(pacote);
      expect(pcm, hasLength(640));
      final dados = ByteData.sublistView(pcm);
      for (var i = 0; i < pcm.length; i += 2) {
        energia += dados.getInt16(i, Endian.little).abs();
      }
    }
    expect(energia, greaterThan(1000000));
  });

  test('chunks parciais e amostras divididas produzem os mesmos pacotes', () {
    final inteiro = CodificadorVoz();
    final fragmentado = CodificadorVoz();
    addTearDown(inteiro.fechar);
    addTearDown(fragmentado.fechar);
    final pcm = tom(6);
    final esperado = inteiro.converter(pcm);
    final obtido = <Uint8List>[];
    var offset = 0;
    for (final tamanho in [1, 638, 2, 741, 3, 1280, 1175]) {
      final fim = min(offset + tamanho, pcm.length);
      obtido.addAll(
        fragmentado.converter(Uint8List.sublistView(pcm, offset, fim)),
      );
      offset = fim;
    }
    obtido.addAll(fragmentado.converter(Uint8List.sublistView(pcm, offset)));
    expect(obtido, esperado);
  });

  test('captura incompleta é descartada ao fechar', () {
    final anterior = CodificadorVoz();
    expect(anterior.converter(Uint8List(639)), isEmpty);
    anterior.fechar();
    final proximo = CodificadorVoz();
    addTearDown(proximo.fechar);
    expect(proximo.converter(Uint8List(1)), isEmpty);
  });

  test('rejeita payload vazio ou fora do limite do protocolo', () {
    final decoder = DecodificadorVoz();
    addTearDown(decoder.fechar);
    expect(() => decoder.converter(Uint8List(0)), throwsFormatException);
    expect(() => decoder.converter(Uint8List(61)), throwsFormatException);
  });
}
