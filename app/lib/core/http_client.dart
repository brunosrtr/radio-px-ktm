import 'package:dio/dio.dart';

import 'env.dart';
import 'rede_local.dart';
import 'token_storage.dart';

/// Cliente HTTP único do app, com interceptor que anexa o Bearer token da
/// sessão atual a toda requisição (exceto `/auth/login`, que não exige
/// token).
class ApiClient {
  ApiClient({Dio? dio, TokenStorage? tokenStorage})
    : _tokenStorage = tokenStorage ?? TokenStorage.instance,
      dio =
          dio ??
          Dio(
            BaseOptions(
              baseUrl: Env.apiBaseUrl,
              connectTimeout: const Duration(seconds: 10),
              receiveTimeout: const Duration(seconds: 10),
            ),
          ) {
    this.dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) async {
          if (dio == null && !RedeLocal.temUrlFixa) {
            try {
              await RedeLocal.instance.assegurarConexao();
              options.baseUrl = Env.apiBaseUrl;
            } catch (erro) {
              handler.reject(
                DioException(
                  requestOptions: options,
                  error: erro,
                  message: 'Computador não encontrado. Confira a rede ou leia o QR Code.',
                ),
              );
              return;
            }
          }
          final token = _tokenStorage.token;
          if (token != null && token.isNotEmpty) {
            options.headers['Authorization'] = 'Bearer $token';
          }
          handler.next(options);
        },
      ),
    );
  }

  final Dio dio;
  final TokenStorage _tokenStorage;
}
