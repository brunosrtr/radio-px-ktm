import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:geolocator/geolocator.dart';
import 'package:radio_px_app/features/canais/criar_canal_page.dart';
import 'package:radio_px_app/features/explorar/explorar_page.dart';
import 'package:radio_px_app/services/canais_api.dart';

class _ApiFalsa extends CanaisApi {
  _ApiFalsa({this.publicos = const []});

  final List<Canal> publicos;
  final chamadasExplorar = <Map<String, Object?>>[];
  final criados = <Map<String, Object?>>[];
  String? conviteUsado;

  @override
  Future<List<Canal>> explorar({String busca = '', double? lat, double? lng, bool perto = false}) async {
    chamadasExplorar.add({'busca': busca, 'lat': lat, 'perto': perto});
    return publicos.where((c) => c.nome.toLowerCase().contains(busca.toLowerCase())).toList();
  }

  @override
  Future<Canal> criar({
    required String nome,
    required String descricao,
    required bool publico,
    required int limite,
    double? latitude,
    double? longitude,
    int? raioMetros,
  }) async {
    criados.add({'nome': nome, 'publico': publico, 'limite': limite, 'raio': raioMetros});
    throw const ErroCanal('teste', 'parou aqui no teste');
  }

  @override
  Future<Canal> entrarPorConvite(String codigo) async {
    conviteUsado = codigo;
    throw const ErroCanal('convite_invalido', 'Convite inválido ou trocado pelo criador. Peça um novo.');
  }
}

Position _posicao() => Position(
  latitude: -23.55,
  longitude: -46.63,
  timestamp: DateTime(2026, 11, 22),
  accuracy: 5,
  altitude: 0,
  altitudeAccuracy: 0,
  heading: 0,
  headingAccuracy: 0,
  speed: 0,
  speedAccuracy: 0,
);

Future<void> _abrir(WidgetTester tester, Widget tela) async {
  await tester.binding.setSurfaceSize(const Size(420, 1000));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(MaterialApp(home: Scaffold(body: tela)));
  await tester.pumpAndSettle();
}

void main() {
  group('código de convite (mesma regra do backend)', () {
    test('aceita código digitado e link do QR', () {
      expect(normalizarCodigoConvite('k7qm 2xpa'), 'K7QM-2XPA');
      expect(normalizarCodigoConvite('K7QM-2XPA'), 'K7QM-2XPA');
      expect(normalizarCodigoConvite('radiopx://app/convite/K7QM2XPA'), 'K7QM-2XPA');
      expect(linkConvite('K7QM-2XPA'), 'radiopx://app/convite/K7QM2XPA');
    });

    test('recusa tamanho errado e caracteres ambíguos', () {
      expect(normalizarCodigoConvite('K7QM-2XP'), isNull);
      expect(normalizarCodigoConvite('K7QM-2XP0'), isNull);
      expect(normalizarCodigoConvite('K7QM-2XPI'), isNull);
    });

    test('canal privado da comunidade', () {
      final c = Canal.deJson({'id': '1', 'nome': 'x', 'comunidade': true, 'visibilidade': 'privado'});
      expect(c.privado, isTrue);
      expect(Canal.deJson({'id': '1', 'nome': 'x', 'comunidade': false, 'visibilidade': 'privado'}).privado, isFalse);
    });
  });

  testWidgets('Explorar lista os públicos perto de mim e busca pelo nome', (tester) async {
    final api = _ApiFalsa(publicos: const [
      Canal(id: '1', nome: 'Serra do Cafezal', participantes: 7, limite: 20, distanciaMetros: 3000, geocerca: true, raioMetros: 15000),
      Canal(id: '2', nome: 'Rodotrem Brasil', participantes: 19, limite: 20),
    ]);
    await _abrir(tester, ExplorarPage(api: api, obterPosicao: () async => _posicao()));

    expect(api.chamadasExplorar.first, {'busca': '', 'lat': -23.55, 'perto': true});
    expect(find.text('Serra do Cafezal'), findsOneWidget);
    expect(find.text('7 de 20 · a 3.0 km · área de 15 km'), findsOneWidget);

    await tester.enterText(find.byKey(const Key('busca-canais')), 'rodo');
    await tester.pump(const Duration(milliseconds: 500));
    await tester.pumpAndSettle();
    expect(find.text('Rodotrem Brasil'), findsOneWidget);
    expect(find.text('Serra do Cafezal'), findsNothing);
  });

  testWidgets('Explorar sem localização mostra todos e avisa', (tester) async {
    final api = _ApiFalsa();
    await _abrir(tester, ExplorarPage(api: api, obterPosicao: () async => null));
    expect(api.chamadasExplorar.first['perto'], isFalse);
    expect(find.text('Sem localização: mostrando todos os canais públicos.'), findsOneWidget);
  });

  testWidgets('entrar com código valida antes de chamar o servidor', (tester) async {
    final api = _ApiFalsa();
    await _abrir(tester, ExplorarPage(api: api, obterPosicao: () async => null));
    await tester.tap(find.text('Entrar com código ou QR de convite'));
    await tester.pumpAndSettle();

    await tester.enterText(find.byKey(const Key('campo-codigo-convite')), 'ABC');
    await tester.tap(find.widgetWithText(FilledButton, 'Entrar'));
    await tester.pumpAndSettle();
    expect(find.textContaining('Código inválido'), findsOneWidget);
    expect(api.conviteUsado, isNull);

    await tester.enterText(find.byKey(const Key('campo-codigo-convite')), 'k7qm2xpa');
    await tester.tap(find.widgetWithText(FilledButton, 'Entrar'));
    await tester.pumpAndSettle();
    expect(api.conviteUsado, 'K7QM-2XPA');
    expect(find.textContaining('Peça um novo'), findsOneWidget);
  });

  testWidgets('Criar canal exige nome e envia privado, limite e área', (tester) async {
    final api = _ApiFalsa();
    await _abrir(tester, CriarCanalPage(api: api, obterPosicao: () async => _posicao()));

    await tester.tap(find.byKey(const Key('botao-criar-canal')));
    await tester.pumpAndSettle();
    expect(find.text('Dê um nome ao canal'), findsOneWidget);
    expect(api.criados, isEmpty);

    await tester.enterText(find.byKey(const Key('campo-nome-canal')), 'Comboio Sul');
    await tester.tap(find.text('15'));
    await tester.tap(find.text('Limitar a uma área'));
    await tester.pumpAndSettle();
    expect(find.text('Raio de 15 km a partir de onde você está'), findsOneWidget);

    await tester.ensureVisible(find.byKey(const Key('botao-criar-canal')));
    await tester.tap(find.byKey(const Key('botao-criar-canal')));
    await tester.pumpAndSettle();
    expect(api.criados.single, {'nome': 'Comboio Sul', 'publico': false, 'limite': 15, 'raio': 15000});
  });
}
