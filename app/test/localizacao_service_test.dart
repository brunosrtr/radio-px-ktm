import 'package:flutter_test/flutter_test.dart';
import 'package:radio_px_app/services/localizacao_service.dart';

void main() {
  group('devePersistirPonto (filtro de deslocamento/intervalo)', () {
    test('persiste ao deslocar 200m ou mais, mesmo com pouco tempo', () {
      expect(
        devePersistirPonto(
          distanciaMetros: 200,
          duracaoDesdeUltimoPonto: const Duration(seconds: 1),
        ),
        isTrue,
      );
    });

    test('persiste após 30s ou mais, mesmo parado', () {
      expect(
        devePersistirPonto(
          distanciaMetros: 0,
          duracaoDesdeUltimoPonto: const Duration(seconds: 30),
        ),
        isTrue,
      );
    });

    test(
      'não persiste quando nem distância nem tempo mínimos foram atingidos',
      () {
        expect(
          devePersistirPonto(
            distanciaMetros: 50,
            duracaoDesdeUltimoPonto: const Duration(seconds: 10),
          ),
          isFalse,
        );
      },
    );

    test('distância abaixo do limiar mas tempo logo acima do limiar', () {
      expect(
        devePersistirPonto(
          distanciaMetros: 199,
          duracaoDesdeUltimoPonto: const Duration(seconds: 31),
        ),
        isTrue,
      );
    });
  });
}
