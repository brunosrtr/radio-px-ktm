import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:radio_px_app/core/http_client.dart';
import 'package:radio_px_app/core/token_storage.dart';
import 'package:radio_px_app/features/auth/auth_service.dart';

/// Adaptador HTTP falso: guarda as requisições e responde com o que o teste
/// definir, sem rede.
class _AdaptadorFalso implements HttpClientAdapter {
  _AdaptadorFalso(this.responder);

  final ({int status, Map<String, dynamic>? corpo}) Function(RequestOptions) responder;
  final requisicoes = <({String metodo, String caminho, Map<String, dynamic>? corpo})>[];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requisicoes.add((
      metodo: options.method,
      caminho: options.path,
      corpo: options.data is Map ? (options.data as Map).cast<String, dynamic>() : null,
    ));
    final resposta = responder(options);
    return ResponseBody.fromString(
      resposta.corpo == null ? '' : jsonEncode(resposta.corpo),
      resposta.status,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

const _sessao = {
  'token': 'jwt-de-teste',
  'usuario': {
    'id': 'u1',
    'nome': 'Maria Souza',
    'apelido': 'Maria da Serra',
    'papel': 'usuario',
    'tem_empresa': false,
  },
};

({AuthService servico, _AdaptadorFalso adaptador}) _montar(
  ({int status, Map<String, dynamic>? corpo}) Function(RequestOptions) responder,
) {
  final adaptador = _AdaptadorFalso(responder);
  final dio = Dio(BaseOptions(baseUrl: 'http://teste'))..httpClientAdapter = adaptador;
  final servico = AuthService(
    apiClient: ApiClient(dio: dio),
    tokenStorage: TokenStorage.instance,
  );
  return (servico: servico, adaptador: adaptador);
}

void main() {
  tearDown(TokenStorage.instance.limpar);

  test('cadastro envia os campos do contrato e guarda a sessão', () async {
    final m = _montar((_) => (status: 201, corpo: _sessao));

    final usuario = await m.servico.cadastrar(
      nome: 'Maria Souza',
      apelido: 'Maria da Serra',
      email: 'maria@exemplo.com',
      senha: 'senha-forte-1',
    );

    final req = m.adaptador.requisicoes.single;
    expect(req.metodo, 'POST');
    expect(req.caminho, '/auth/cadastro');
    expect(req.corpo, containsPair('email', 'maria@exemplo.com'));
    expect(req.corpo, containsPair('apelido', 'Maria da Serra'));
    expect(req.corpo, containsPair('aceitou_termos_versao', versaoTermosAtual));
    expect(usuario.temEmpresa, isFalse);
    expect(TokenStorage.instance.token, 'jwt-de-teste');
    expect(TokenStorage.instance.usuario?.apelido, 'Maria da Serra');
  });

  test('e-mail já cadastrado vira ErroConta com mensagem amigável', () async {
    final m = _montar(
      (_) => (status: 409, corpo: {'codigo': 'email_em_uso', 'mensagem': 'este e-mail já tem conta'}),
    );

    await expectLater(
      m.servico.cadastrar(nome: 'Maria', apelido: 'Maria', email: 'maria@exemplo.com', senha: 'senha-forte-1'),
      throwsA(isA<ErroConta>()
          .having((e) => e.codigo, 'codigo', 'email_em_uso')
          .having((e) => e.mensagem, 'mensagem', contains('já tem conta'))),
    );
    expect(TokenStorage.instance.token, isNull);
  });

  test('login envia identificador e registra o dispositivo', () async {
    final m = _montar((_) => (status: 200, corpo: _sessao));

    await m.servico.login('maria@exemplo.com', 'senha-forte-1');

    final req = m.adaptador.requisicoes.single;
    expect(req.caminho, '/auth/login');
    expect(req.corpo, containsPair('identificador', 'maria@exemplo.com'));
    expect(req.corpo?['dispositivo'], isA<Map>());
  });

  test('excluir conta chama DELETE /me e encerra a sessão', () async {
    final m = _montar((_) => (status: 204, corpo: null));
    TokenStorage.instance.definir('jwt-antigo');

    await m.servico.excluirConta();

    expect(m.adaptador.requisicoes.single.metodo, 'DELETE');
    expect(m.adaptador.requisicoes.single.caminho, '/me');
    expect(TokenStorage.instance.token, isNull);
  });

  test('código de recuperação inválido vira mensagem para pedir outro', () async {
    final m = _montar((_) => (status: 400, corpo: {'codigo': 'codigo_invalido', 'mensagem': 'x'}));

    await expectLater(
      m.servico.redefinirSenha(email: 'maria@exemplo.com', codigo: '123456', novaSenha: 'nova-senha-1'),
      throwsA(isA<ErroConta>().having((e) => e.mensagem, 'mensagem', contains('Peça um novo código'))),
    );
  });
}
