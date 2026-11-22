import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:hive_flutter/hive_flutter.dart';
import 'package:radio_px_app/core/router.dart';
import 'package:radio_px_app/core/sessao_app.dart';
import 'package:radio_px_app/core/token_storage.dart';
import 'package:radio_px_app/main.dart';
import 'package:radio_px_app/models/usuario_sessao.dart';

void main() {
  setUpAll(() => Hive.init(Directory.systemTemp.createTempSync('radiopx_teste').path));

  setUp(() {
    TokenStorage.instance.limpar();
    SessaoApp.instance
      ..apresentacaoVista = false
      ..permissoesVistas = false;
  });

  Future<void> abrir(WidgetTester tester, {String? inicial}) async {
    await tester.binding.setSurfaceSize(const Size(400, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    await tester.pumpWidget(RadioPxApp(router: criarRouter(inicial: inicial)));
    await tester.pumpAndSettle();
  }

  testWidgets('primeiro acesso: apresentação em 3 telas e depois o cadastro', (tester) async {
    await abrir(tester);
    expect(find.text('A estrada inteira\nno mesmo canal.'), findsOneWidget);

    await tester.tap(find.byKey(const Key('boas-vindas-proximo')));
    await tester.pumpAndSettle();
    expect(find.text('REDE ORBIT'), findsOneWidget);

    await tester.tap(find.byKey(const Key('boas-vindas-proximo')));
    await tester.pumpAndSettle();
    expect(find.text('Criar minha conta'), findsOneWidget);

    await tester.tap(find.byKey(const Key('boas-vindas-proximo')));
    await tester.pumpAndSettle();
    expect(find.text('Entre para a rede da estrada.'), findsOneWidget);
    expect(SessaoApp.instance.apresentacaoVista, isTrue);
  });

  testWidgets('quem já viu a apresentação cai no login e abre a página Sobre', (tester) async {
    SessaoApp.instance.apresentacaoVista = true;
    await abrir(tester);
    expect(find.text('E-mail ou login'), findsOneWidget);
    expect(find.text('Criar conta grátis'), findsOneWidget);

    final sobre = find.text('Conheça a KTM Indústria');
    await tester.ensureVisible(sobre);
    await tester.tap(sobre);
    await tester.pumpAndSettle();
    expect(find.text('Sobre a KTM'), findsOneWidget);
  });

  testWidgets('logado: as 4 abas ficam a um toque', (tester) async {
    TokenStorage.instance.definir(
      'token-de-teste',
      usuario: const UsuarioSessao(id: 'u1', apelido: 'Maria da Serra', papel: 'usuario', temEmpresa: false, email: 'maria@exemplo.com'),
    );
    SessaoApp.instance
      ..apresentacaoVista = true
      ..permissoesVistas = true;
    await abrir(tester, inicial: '/perfil');

    expect(find.byType(NavigationDestination), findsNWidgets(4));
    expect(find.text('Maria da Serra'), findsOneWidget);
    expect(find.text('Modo Orbit virtual'), findsNothing, reason: 'conta comum não vê o modo de demonstração');

    // As outras abas buscam dados no servidor; aqui só confirmamos que um
    // toque troca de área (FR-038).
    await tester.tap(find.text('Contribuição'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(find.text('Sua contribuição'), findsOneWidget);

    // Rota de entrada com sessão aberta volta para os canais.
    expect(decidirRota(destino: '/entrar', logado: true, apresentacaoVista: true, permissoesVistas: true), '/canais');
  });
}
