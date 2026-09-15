/// Guarda o JWT da sessão atual em memória, para o interceptor de
/// autenticação do [ApiClient] e para a conexão WebSocket do
/// `canal_service.dart` (fase de User Story 1).
class TokenStorage {
  TokenStorage._();

  static final TokenStorage instance = TokenStorage._();

  String? _token;

  String? get token => _token;

  void definir(String token) => _token = token;

  void limpar() => _token = null;
}
