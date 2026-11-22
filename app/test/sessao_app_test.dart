import 'package:flutter_test/flutter_test.dart';
import 'package:radio_px_app/core/sessao_app.dart';

void main() {
  String? rota(String destino, {bool logado = false, bool apresentacao = false, bool permissoes = false}) =>
      decidirRota(destino: destino, logado: logado, apresentacaoVista: apresentacao, permissoesVistas: permissoes);

  test('sem conta, primeira vez: apresentação', () {
    expect(rota('/canais'), '/boas-vindas');
    expect(rota('/boas-vindas'), isNull);
    expect(rota('/cadastro'), isNull);
  });

  test('sem conta, já viu a apresentação: login', () {
    expect(rota('/canais', apresentacao: true), '/entrar');
    expect(rota('/perfil', apresentacao: true), '/entrar');
    expect(rota('/sobre', apresentacao: true), isNull);
  });

  test('logado sem ter visto as permissões: permissões primeiro', () {
    expect(rota('/canais', logado: true, apresentacao: true), '/permissoes');
    expect(rota('/permissoes', logado: true, apresentacao: true), isNull);
  });

  test('logado: telas de entrada levam aos canais; o resto segue', () {
    for (final entrada in ['/entrar', '/cadastro', '/boas-vindas', '/permissoes']) {
      expect(rota(entrada, logado: true, apresentacao: true, permissoes: true), '/canais');
    }
    expect(rota('/contribuicao', logado: true, apresentacao: true, permissoes: true), isNull);
    expect(rota('/canal/abc', logado: true, apresentacao: true, permissoes: true), isNull);
  });
}
