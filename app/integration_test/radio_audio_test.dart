// Teste explícito em aparelho: aciona o microfone por aproximadamente 2s.
// Exige backend de desenvolvimento com motorista1/motorista2 e canal sem geocerca.
// Somente contagens são verificadas; nenhum áudio é gravado em arquivo/log.
import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:permission_handler/permission_handler.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:web_socket_channel/web_socket_channel.dart';
import 'package:radio_px_app/main.dart' as app;
import 'package:radio_px_app/core/env.dart';
import 'package:radio_px_app/services/codec_voz.dart';

Future<void> aguardar(WidgetTester tester, bool Function() pronto) async {
  final limite = DateTime.now().add(const Duration(seconds: 20));
  while (!pronto() && DateTime.now().isBefore(limite)) {
    await tester.pump(const Duration(milliseconds: 100));
  }
  expect(pronto(), isTrue, reason: 'Condição não atingida em 20 segundos.');
}

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  testWidgets('PTT envia Opus pelo backend e recebe voz de outra conta', (
    tester,
  ) async {
    final permissoes = await [
      Permission.microphone,
      Permission.locationWhenInUse,
      Permission.notification,
    ].request();
    expect(permissoes[Permission.microphone]?.isGranted, isTrue);
    expect(permissoes[Permission.locationWhenInUse]?.isGranted, isTrue);
    final api = Dio(BaseOptions(baseUrl: Env.apiBaseUrl));
    final login = await api.post(
      '/auth/login',
      data: {'login': 'motorista2', 'senha': 'motorista123'},
    );
    final token = login.data['token'] as String;
    api.options.headers['Authorization'] = 'Bearer $token';
    final canais = (await api.get('/canais')).data['canais'] as List;
    final canal = canais.firstWhere((c) => c['geocerca_ativa'] == false);
    final remoto = WebSocketChannel.connect(
      Uri.parse('${Env.wsBaseUrl}/ws')
          .replace(queryParameters: {'token': token}),
    );
    await remoto.ready;
    var entrou = false;
    var pacotesRecebidos = 0;
    var amostrasRecebidas = 0;
    String? slot;
    DecodificadorVoz? decoder;
    final assinatura = remoto.stream.listen((evento) {
      if (evento is List<int>) {
        final pcm = decoder!.converter(Uint8List.fromList(evento));
        pacotesRecebidos++;
        amostrasRecebidas += pcm.length ~/ 2;
      } else {
        final mensagem = jsonDecode(evento as String);
        switch (mensagem['tipo']) {
          case 'canal_entrado':
            entrou = true;
          case 'inicio_reproducao':
            decoder?.fechar();
            decoder = DecodificadorVoz();
          case 'fim_reproducao':
            decoder?.fechar();
            decoder = null;
          case 'slot_concedido':
            slot = mensagem['dados']['transmissao_id'] as String;
        }
      }
    });
    void enviar(String tipo, Map<String, dynamic> dados) =>
        remoto.sink.add(jsonEncode({'tipo': tipo, 'dados': dados}));
    addTearDown(() async {
      enviar('sair_canal', {'canal_id': canal['id']});
      await remoto.sink.close();
      await assinatura.cancel();
      decoder?.fechar();
      api.close();
    });
    enviar('entrar_canal', {'canal_id': canal['id']});
    await aguardar(tester, () => entrou);
    await app.main();
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextFormField).at(0), 'motorista1');
    await tester.enterText(find.byType(TextFormField).at(1), 'motorista123');
    await tester.tap(find.text('Entrar no Rádio PX'));
    await aguardar(
      tester,
      () => find.text(canal['nome'] as String).evaluate().isNotEmpty,
    );
    await tester.tap(find.text(canal['nome'] as String).first);
    await aguardar(
      tester,
      () => find.text('Segure para falar').evaluate().isNotEmpty,
    );
    // Duas capturas verificam também fechamento e reabertura do gravador.
    for (var tentativa = 0; tentativa < 2; tentativa++) {
      final anteriores = pacotesRecebidos;
      final gesto = await tester.startGesture(
        tester.getCenter(find.byIcon(Icons.mic)),
      );
      await aguardar(
        tester,
        () => find.text('Transmitindo…').evaluate().isNotEmpty,
      );
      await tester.pump(const Duration(seconds: 1));
      await aguardar(tester, () => pacotesRecebidos > anteriores + 5);
      await gesto.up();
      await aguardar(
        tester,
        () => find.text('Segure para falar').evaluate().isNotEmpty,
      );
    }
    expect(amostrasRecebidas, greaterThan(3200));
    enviar('solicitar_slot', {'canal_id': canal['id']});
    await aguardar(tester, () => slot != null);
    final encoder = CodificadorVoz();
    try {
      for (var frame = 0; frame < 50; frame++) {
        final pcm = ByteData(640);
        for (var i = 0; i < 320; i++) {
          pcm.setInt16(
            i * 2,
            (sin(2 * pi * 440 * (frame * 320 + i) / 16000) * 4000).round(),
            Endian.little,
          );
        }
        remoto.sink.add(encoder.converter(pcm.buffer.asUint8List()).single);
        await tester.pump(const Duration(milliseconds: 20));
      }
      expect(find.text('Motorista Dois'), findsOneWidget);
      enviar('finalizar_transmissao', {'transmissao_id': slot});
      await aguardar(
        tester,
        () => find.text('Canal em silêncio').evaluate().isNotEmpty,
      );
      await tester.pump(const Duration(seconds: 2));
      expect(find.textContaining('Não foi possível'), findsNothing);
      expect(tester.takeException(), isNull);
    } finally {
      encoder.fechar();
    }
    await tester.pageBack();
    await tester.pump(const Duration(seconds: 2));
    expect(find.text('Canais'), findsOneWidget);
  });
}
