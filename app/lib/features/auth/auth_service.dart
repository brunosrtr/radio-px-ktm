import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart' show TargetPlatform, defaultTargetPlatform;

import '../../core/env.dart';
import '../../core/http_client.dart';
import '../../core/token_storage.dart';
import '../../models/usuario_sessao.dart';

/// Versão dos termos de uso e da política de privacidade aceita no cadastro.
/// Precisa ser a mesma servida pelo backend em `/termos` e `/privacidade`.
const versaoTermosAtual = '2026-11';

/// Erro de negócio devolvido pela API de contas, com o `codigo` do contrato
/// (`email_em_uso`, `dados_invalidos`, `muitas_tentativas`, ...) e uma
/// mensagem pronta para mostrar ao usuário.
class ErroConta implements Exception {
  const ErroConta(this.codigo, this.mensagem);

  final String codigo;
  final String mensagem;

  @override
  String toString() => mensagem;
}

/// Autentica o usuário, cria contas do público e cuida da recuperação e da
/// exclusão da conta. Guarda o token e o usuário da sessão no
/// [TokenStorage] para o restante do app.
class AuthService {
  AuthService({ApiClient? apiClient, TokenStorage? tokenStorage})
    : _apiClient = apiClient ?? ApiClient(),
      _tokenStorage = tokenStorage ?? TokenStorage.instance;

  final ApiClient _apiClient;
  final TokenStorage _tokenStorage;

  Map<String, dynamic> get _dispositivo => {
    'plataforma': defaultTargetPlatform == TargetPlatform.iOS ? 'ios' : 'android',
    'versao_app': Env.appVersion,
  };

  /// Entra com e-mail (contas do público) ou CPF/login (contas de empresa).
  /// Registra o dispositivo atual (FR-017) no mesmo request.
  Future<UsuarioSessao> login(String identificador, String senha) {
    return _criarSessao('/auth/login', {
      'identificador': identificador,
      'senha': senha,
      'dispositivo': _dispositivo,
    });
  }

  /// Cria a conta do público e já entra com ela (FR-023).
  Future<UsuarioSessao> cadastrar({
    required String nome,
    required String apelido,
    required String email,
    required String senha,
  }) {
    return _criarSessao('/auth/cadastro', {
      'nome': nome,
      'apelido': apelido,
      'email': email,
      'senha': senha,
      'aceitou_termos_versao': versaoTermosAtual,
      'dispositivo': _dispositivo,
    });
  }

  /// Pede o código de recuperação de senha. A resposta é a mesma exista ou
  /// não a conta.
  Future<void> solicitarRecuperacao(String email) =>
      _executar(() => _apiClient.dio.post('/auth/recuperar-senha', data: {'email': email}));

  /// Troca a senha usando o código recebido por e-mail.
  Future<void> redefinirSenha({
    required String email,
    required String codigo,
    required String novaSenha,
  }) => _executar(
    () => _apiClient.dio.post(
      '/auth/redefinir-senha',
      data: {'email': email, 'codigo': codigo, 'nova_senha': novaSenha},
    ),
  );

  /// Recarrega os dados da conta da sessão atual.
  Future<UsuarioSessao> me() async {
    final resposta = await _executar(() => _apiClient.dio.get('/me'));
    final usuario = UsuarioSessao.deJson(
      (resposta.data as Map).cast<String, dynamic>(),
    );
    _tokenStorage.definirUsuario(usuario);
    return usuario;
  }

  /// Altera nome e/ou apelido (PATCH /me); devolve a conta atualizada.
  Future<UsuarioSessao> atualizarPerfil({String? nome, String? apelido}) async {
    final resposta = await _executar(
      () => _apiClient.dio.patch('/me', data: {'nome': ?nome, 'apelido': ?apelido}),
    );
    final usuario = UsuarioSessao.deJson((resposta.data as Map).cast<String, dynamic>());
    _tokenStorage.definirUsuario(usuario);
    return usuario;
  }

  /// Exclui a conta e encerra a sessão (FR-026).
  Future<void> excluirConta() async {
    await _executar(() => _apiClient.dio.delete('/me'));
    sair();
  }

  void sair() => _tokenStorage.limpar();

  Future<UsuarioSessao> _criarSessao(String rota, Map<String, dynamic> corpo) async {
    final resposta = await _executar(() => _apiClient.dio.post(rota, data: corpo));
    final dados = (resposta.data as Map).cast<String, dynamic>();
    final usuario = UsuarioSessao.deJson(
      (dados['usuario'] as Map).cast<String, dynamic>(),
    );
    _tokenStorage.definir(dados['token'] as String, usuario: usuario);
    return usuario;
  }

  /// Converte respostas de erro da API em [ErroConta] e falhas de rede numa
  /// mensagem compreensível.
  Future<Response<dynamic>> _executar(
    Future<Response<dynamic>> Function() chamada,
  ) async {
    try {
      return await chamada();
    } on DioException catch (e) {
      final dados = e.response?.data;
      if (dados is Map && dados['codigo'] is String) {
        throw ErroConta(
          dados['codigo'] as String,
          mensagemAmigavel(dados['codigo'] as String, dados['mensagem'] as String?),
        );
      }
      throw ErroConta(
        'sem_conexao',
        e.message ?? 'Não foi possível falar com o servidor. Confira sua conexão.',
      );
    }
  }
}

/// Mensagens em linguagem simples para os códigos de erro de conta.
String mensagemAmigavel(String codigo, String? mensagemServidor) {
  switch (codigo) {
    case 'email_em_uso':
      return 'Este e-mail já tem conta. Entre com ele ou recupere a senha.';
    case 'credenciais_invalidas':
      return 'E-mail, login ou senha incorretos.';
    case 'muitas_tentativas':
      return 'Muitas tentativas seguidas. Aguarde alguns minutos e tente de novo.';
    case 'codigo_invalido':
      return 'Código inválido ou expirado. Peça um novo código.';
    case 'recuperacao_indisponivel':
      return 'A recuperação de senha não está disponível neste servidor.';
    case 'dados_invalidos':
      return _primeiraMaiuscula(mensagemServidor ?? 'Confira os dados informados.');
    default:
      return mensagemServidor ?? 'Algo deu errado. Tente de novo.';
  }
}

String _primeiraMaiuscula(String texto) =>
    texto.isEmpty ? texto : texto[0].toUpperCase() + texto.substring(1);
