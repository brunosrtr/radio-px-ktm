import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:radio_px_app/data/fila_leituras_local.dart';
import 'package:radio_px_app/orbit/leitor_orbit.dart';
import 'package:radio_px_app/orbit/leituras_service.dart';

class _FilaMemoria implements FilaLeituras {
  final itens = <int, LeituraLocal>{};
  var _proxima = 0;

  @override
  Future<void> abrir() async {}

  @override
  Future<void> adicionar(LeituraLocal leitura) async => itens[_proxima++] = leitura;

  @override
  Map<dynamic, LeituraLocal> listarPendentes({int limite = 100}) =>
      Map.fromEntries(itens.entries.take(limite));

  @override
  Future<void> removerConfirmados(Iterable<dynamic> chaves) async => chaves.forEach(itens.remove);

  @override
  Future<int> descartarAntigas(DateTime agora) async => 0;

  @override
  int get tamanho => itens.length;
}

class _EnviadorFalso implements EnviadorLeituras {
  final lotes = <List<Map<String, dynamic>>>[];
  Object? falha;
  var sinais = 0;

  @override
  Future<void> enviarLote(List<Map<String, dynamic>> leituras) async {
    if (falha != null) throw falha!;
    lotes.add(leituras);
  }

  @override
  Future<void> enviarSinal() async => sinais++;
}

void main() {
  final t0 = DateTime(2026, 11, 21, 12);
  late _FilaMemoria fila;
  late _EnviadorFalso enviador;
  late PosicaoLeitura? posicao;
  late int bips;
  late LeiturasService servico;

  setUp(() {
    fila = _FilaMemoria();
    enviador = _EnviadorFalso();
    posicao = PosicaoLeitura(latitude: -23.55, longitude: -46.63, precisaoMetros: 8, capturadaEm: t0);
    bips = 0;
    servico = LeiturasService(
      fila: fila,
      obterPosicao: () async => posicao,
      enviador: enviador,
      aoRegistrar: () async => bips++,
      relogio: () => t0,
    );
  });

  DeteccaoOrbit deteccao(String id, Duration depois) =>
      DeteccaoOrbit(orbitId: id, rssi: -60, instante: t0.add(depois));

  test('cada detecção aceita vira leitura com a posição do celular e um bip', () async {
    expect(await servico.processar(deteccao('ORBIT-0A1B2C', Duration.zero)), isTrue);
    final leitura = fila.itens.values.single;
    expect(leitura.orbitId, 'ORBIT-0A1B2C');
    expect(leitura.latitude, -23.55);
    expect(leitura.rssi, -60);
    expect(bips, 1);
    expect(servico.leiturasSessao.value, 1);
  });

  test('o mesmo Orbit no alcance não gera leitura a cada anúncio', () async {
    await servico.processar(deteccao('ORBIT-0A1B2C', Duration.zero));
    await servico.processar(deteccao('ORBIT-0A1B2C', const Duration(seconds: 2)));
    await servico.processar(deteccao('ORBIT-0A1B2C', const Duration(seconds: 10)));
    expect(fila.tamanho, 1);
    expect(bips, 1);
  });

  test('sem posição recente a leitura é descartada', () async {
    posicao = null;
    expect(await servico.processar(deteccao('ORBIT-0A1B2C', Duration.zero)), isFalse);
    posicao = PosicaoLeitura(latitude: -23.5, longitude: -46.6, capturadaEm: t0.subtract(const Duration(minutes: 5)));
    expect(await servico.processar(deteccao('ORBIT-0A1B2D', Duration.zero)), isFalse);
    expect(fila.tamanho, 0);
    expect(bips, 0);
  });

  test('envia em lotes de até 100 e só remove o que foi confirmado', () async {
    for (var i = 0; i < 150; i++) {
      await fila.adicionar(LeituraLocal(
        orbitId: 'ORBIT-0A1B2C', latitude: -23.5, longitude: -46.6, detectadoEm: t0.add(Duration(seconds: i)),
      ));
    }
    await servico.enviarPendentes();
    expect(enviador.lotes.map((l) => l.length), [100, 50]);
    expect(fila.tamanho, 0);
  });

  test('sem conexão as leituras continuam na fila, com o horário real', () async {
    await servico.processar(deteccao('ORBIT-0A1B2C', Duration.zero));
    enviador.falha = Exception('sem rede');
    await servico.enviarPendentes();
    expect(fila.tamanho, 1);

    enviador.falha = null;
    await servico.enviarPendentes();
    expect(fila.tamanho, 0);
    expect(enviador.lotes.single.single['detectado_em'], t0.toUtc().toIso8601String());
  });

  test('lote recusado pelo servidor (422) é descartado para não travar a fila', () async {
    await servico.processar(deteccao('ORBIT-0A1B2C', Duration.zero));
    enviador.falha = const LoteRecusado('horário no futuro');
    await servico.enviarPendentes();
    expect(fila.tamanho, 0);
  });

  test('sem consentimento no servidor, a fila é apagada sem enviar', () async {
    var avisado = false;
    final semConsentimento = LeiturasService(
      fila: fila,
      obterPosicao: () async => posicao,
      enviador: _EnviadorSemConsentimento(),
      relogio: () => t0,
      aoPerderConsentimento: () => avisado = true,
    );
    await semConsentimento.processar(deteccao('ORBIT-0A1B2C', Duration.zero));
    await semConsentimento.processar(deteccao('ORBIT-0A1B2D', Duration.zero));
    expect(fila.tamanho, 2);

    await semConsentimento.enviarPendentes();
    expect(fila.tamanho, 0);
    expect(avisado, isTrue);
  });

  test('ao revogar, descartarPendentes apaga a fila e parar não envia', () async {
    await servico.processar(deteccao('ORBIT-0A1B2C', Duration.zero));
    await servico.parar(enviarPendentes: false);
    await servico.descartarPendentes();
    expect(fila.tamanho, 0);
    expect(enviador.lotes, isEmpty);
  });

  test('ignora o próprio Orbit virtual', () async {
    servico.deduplicador.orbitProprio = 'ORBIT-0A1B2C';
    expect(await servico.processar(deteccao('ORBIT-0A1B2C', Duration.zero)), isFalse);
    expect(await servico.processar(deteccao('ORBIT-0A1B2D', Duration.zero)), isTrue);
  });

  test('iniciar escuta o leitor e manda o sinal de vida', () async {
    final controlador = StreamController<DeteccaoOrbit>();
    await servico.iniciar(controlador.stream);
    controlador.add(deteccao('ORBIT-0A1B2C', Duration.zero));
    await Future<void>.delayed(Duration.zero);
    await Future<void>.delayed(Duration.zero);
    expect(fila.tamanho + enviador.lotes.length, greaterThan(0));
    expect(enviador.sinais, 1);
    await servico.parar();
    await controlador.close();
  });
}

class _EnviadorSemConsentimento implements EnviadorLeituras {
  @override
  Future<void> enviarLote(List<Map<String, dynamic>> leituras) async => throw const SemConsentimento();

  @override
  Future<void> enviarSinal() async {}
}
