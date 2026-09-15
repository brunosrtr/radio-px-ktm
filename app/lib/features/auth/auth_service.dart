import '../../core/http_client.dart';
import '../../core/token_storage.dart';

/// Autentica o usuário e guarda o token da sessão para o restante do app
/// (interceptor do [ApiClient] e conexão WebSocket do `canal_service.dart`).
class AuthService {
  AuthService({ApiClient? apiClient, TokenStorage? tokenStorage})
    : _apiClient = apiClient ?? ApiClient(),
      _tokenStorage = tokenStorage ?? TokenStorage.instance;

  final ApiClient _apiClient;
  final TokenStorage _tokenStorage;

  Future<void> login(String login, String senha) async {
    final resposta = await _apiClient.dio.post(
      '/auth/login',
      data: {'login': login, 'senha': senha},
    );
    _tokenStorage.definir(resposta.data['token'] as String);
  }
}
