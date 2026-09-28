import 'package:flutter/foundation.dart' show TargetPlatform, defaultTargetPlatform;

import '../../core/env.dart';
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

  /// Autentica e registra o dispositivo atual (FR-017) no mesmo request.
  Future<void> login(String login, String senha) async {
    final resposta = await _apiClient.dio.post(
      '/auth/login',
      data: {
        'login': login,
        'senha': senha,
        'dispositivo': {
          'plataforma': defaultTargetPlatform == TargetPlatform.iOS ? 'ios' : 'android',
          'versao_app': Env.appVersion,
        },
      },
    );
    _tokenStorage.definir(resposta.data['token'] as String);
  }
}
