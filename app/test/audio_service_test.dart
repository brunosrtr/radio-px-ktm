import 'package:flutter/foundation.dart' show TargetPlatform;
import 'package:flutter_test/flutter_test.dart';
import 'package:radio_px_app/services/audio_service.dart';

void main() {
  group('bytesAceitosPeloPlayer', () {
    test('no Android o retorno 1 significa o bloco inteiro, não 1 byte', () {
      expect(
        bytesAceitosPeloPlayer(
          retorno: 1,
          restantes: 640,
          plataforma: TargetPlatform.android,
        ),
        640,
      );
    });

    test('no iOS usa a contagem de bytes devolvida pelo player', () {
      expect(
        bytesAceitosPeloPlayer(
          retorno: 320,
          restantes: 640,
          plataforma: TargetPlatform.iOS,
        ),
        320,
      );
    });

    test('recusa retorno que desalinharia amostras PCM16', () {
      expect(
        () => bytesAceitosPeloPlayer(
          retorno: 1,
          restantes: 640,
          plataforma: TargetPlatform.iOS,
        ),
        throwsStateError,
      );
    });

    test('recusa quando o player não aceitou nada', () {
      for (final plataforma in [TargetPlatform.android, TargetPlatform.iOS]) {
        expect(
          () => bytesAceitosPeloPlayer(
            retorno: 0,
            restantes: 640,
            plataforma: plataforma,
          ),
          throwsStateError,
        );
      }
    });
  });
}
