import 'package:flutter_test/flutter_test.dart';
import 'package:radio_px_app/orbit/deduplicador_leituras.dart';

void main() {
  final t0 = DateTime(2026, 11, 21, 12);

  test('no máximo uma leitura por Orbit a cada 30 s', () {
    final d = DeduplicadorLeituras();
    expect(d.aceitar('ORBIT-0A1B2C', t0), isTrue);
    expect(d.aceitar('ORBIT-0A1B2C', t0.add(const Duration(seconds: 1))), isFalse);
    expect(d.aceitar('ORBIT-0A1B2C', t0.add(const Duration(seconds: 29))), isFalse);
    expect(d.aceitar('ORBIT-0A1B2C', t0.add(const Duration(seconds: 30))), isTrue);
  });

  test('Orbits diferentes não se bloqueiam', () {
    final d = DeduplicadorLeituras();
    expect(d.aceitar('ORBIT-0A1B2C', t0), isTrue);
    expect(d.aceitar('ORBIT-0A1B2D', t0), isTrue);
    expect(d.aceitar('ORBIT-0A1B2D', t0.add(const Duration(seconds: 5))), isFalse);
  });

  test('ignora o próprio Orbit virtual', () {
    final d = DeduplicadorLeituras()..orbitProprio = 'ORBIT-0A1B2C';
    expect(d.aceitar('ORBIT-0A1B2C', t0), isFalse);
    expect(d.aceitar('ORBIT-0A1B2D', t0), isTrue);
    d.orbitProprio = null;
    expect(d.aceitar('ORBIT-0A1B2C', t0), isTrue);
  });
}
