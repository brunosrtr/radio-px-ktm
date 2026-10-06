import 'package:flutter_test/flutter_test.dart';
import 'package:radio_px_app/main.dart';

void main() {
  testWidgets('Login KTM abre a página Sobre nós', (WidgetTester tester) async {
    await tester.pumpWidget(const RadioPxApp());
    expect(find.text('KTM · RÁDIO PX'), findsOneWidget);
    expect(find.text('CPF ou login'), findsOneWidget);
    expect(find.text('Entrar no Rádio PX'), findsOneWidget);
    final sobre = find.text('Conheça a KTM Indústria');
    await tester.ensureVisible(sobre);
    await tester.tap(sobre);
    await tester.pumpAndSettle();
    expect(find.text('Sobre a KTM'), findsOneWidget);
    expect(find.text('https://ktm.ind.br/'), findsOneWidget);
  });
}
