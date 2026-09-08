import 'package:flutter_test/flutter_test.dart';

import 'package:radio_px_app/main.dart';

void main() {
  testWidgets('App renders the placeholder home screen', (WidgetTester tester) async {
    await tester.pumpWidget(const RadioPxApp());

    expect(find.text('Rádio PX Digital'), findsWidgets);
  });
}
