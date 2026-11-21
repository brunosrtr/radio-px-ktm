import '../models/usuario_sessao.dart';

/// Guarda o JWT e os dados do usuário da sessão atual em memória, para o
/// interceptor de autenticação do [ApiClient], a conexão WebSocket do
/// `canal_service.dart` e as telas que dependem do tipo de conta.
class TokenStorage {
  TokenStorage._();

  static final TokenStorage instance = TokenStorage._();

  String? _token;
  UsuarioSessao? _usuario;

  String? get token => _token;

  UsuarioSessao? get usuario => _usuario;

  void definir(String token, {UsuarioSessao? usuario}) {
    _token = token;
    if (usuario != null) _usuario = usuario;
  }

  void definirUsuario(UsuarioSessao usuario) => _usuario = usuario;

  void limpar() {
    _token = null;
    _usuario = null;
  }
}
