import 'package:flutter_test/flutter_test.dart';
import 'package:radio_px_app/orbit/orbit_sinal.dart';

void main() {
  test('codifica e lê o mesmo Orbit (ida e volta)', () {
    final carga = OrbitSinal.codificarCarga('ORBIT-0A1B2C');
    expect(carga, [0x01, 0x01, 0x0A, 0x1B, 0x2C, 0, 0, 0]);
    expect(OrbitSinal.lerCarga(carga), 'ORBIT-0A1B2C');
  });

  test('aceita Orbit real (tipo 0x02) e ignora os bytes reservados', () {
    expect(OrbitSinal.lerCarga([0x01, 0x02, 0xFF, 0x00, 0x7E, 9, 9, 9]), 'ORBIT-FF007E');
  });

  test('recusa carga curta, versão ou tipo desconhecidos', () {
    expect(OrbitSinal.lerCarga(null), isNull);
    expect(OrbitSinal.lerCarga([0x01, 0x01, 0x0A]), isNull);
    expect(OrbitSinal.lerCarga([0x02, 0x01, 0x0A, 0x1B, 0x2C, 0, 0, 0]), isNull);
    expect(OrbitSinal.lerCarga([0x01, 0x09, 0x0A, 0x1B, 0x2C, 0, 0, 0]), isNull);
  });

  test('o ID segue o mesmo formato validado pelo backend', () {
    expect(OrbitSinal.idValido('ORBIT-0A1B2C'), isTrue);
    expect(OrbitSinal.idValido('ORBIT-0a1b2c'), isFalse);
    expect(OrbitSinal.idValido('ORBIT-0A1B2'), isFalse);
    expect(() => OrbitSinal.codificarCarga('caminhão'), throwsArgumentError);
  });
}
