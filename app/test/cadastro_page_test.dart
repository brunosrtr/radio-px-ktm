import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:radio_px_app/features/auth/cadastro_page.dart';
import 'package:radio_px_app/features/auth/validacoes.dart';

Future<void> _abrir(WidgetTester tester) async {
  await tester.binding.setSurfaceSize(const Size(500, 1400));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(const MaterialApp(home: CadastroPage()));
}

Finder _campo(String rotulo) => find.widgetWithText(TextFormField, rotulo);

void main() {
  testWidgets('botão fica desabilitado até aceitar os termos', (tester) async {
    await _abrir(tester);

    FilledButton botao() => tester.widget<FilledButton>(find.byKey(const Key('botao-criar-conta')));
    expect(botao().onPressed, isNull);

    await tester.tap(find.byKey(const Key('aceite-termos')));
    await tester.pump();
    expect(botao().onPressed, isNotNull);
  });

  testWidgets('mostra erros de apelido curto e senhas diferentes', (tester) async {
    await _abrir(tester);

    await tester.enterText(_campo('Nome completo'), 'Maria Souza');
    await tester.enterText(_campo('Apelido de rádio'), 'M');
    await tester.enterText(_campo('E-mail'), 'maria@exemplo.com');
    await tester.enterText(_campo('Senha'), 'senha-forte-1');
    await tester.enterText(_campo('Confirme a senha'), 'outra-senha-1');
    await tester.tap(find.byKey(const Key('aceite-termos')));
    await tester.pump();
    await tester.tap(find.byKey(const Key('botao-criar-conta')));
    await tester.pump();

    expect(find.text('O apelido deve ter de 2 a 24 caracteres'), findsOneWidget);
    expect(find.text('As senhas não conferem'), findsOneWidget);
  });

  group('validações iguais às do backend', () {
    test('apelido', () {
      expect(validarApelido('M'), isNotNull);
      expect(validarApelido('Ma'), isNull);
      expect(validarApelido('é' * 24), isNull);
      expect(validarApelido('a' * 25), isNotNull);
    });

    test('e-mail', () {
      expect(validarEmail('maria@exemplo.com'), isNull);
      expect(validarEmail('maria@exemplo'), isNotNull);
      expect(validarEmail('maria.exemplo.com'), isNotNull);
    });

    test('senha', () {
      expect(validarSenha('1234567'), isNotNull);
      expect(validarSenha('12345678'), isNull);
    });
  });
}
