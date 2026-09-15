/// Configuração de ambiente do app, resolvida em tempo de build via
/// `--dart-define` (sem arquivo `.env` embutido no bundle do app).
class Env {
  Env._();

  static const String apiBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://localhost:8080',
  );

  static const String wsBaseUrl = String.fromEnvironment(
    'WS_BASE_URL',
    defaultValue: 'ws://localhost:8080',
  );

  /// Espelha a versão em pubspec.yaml — enviada no login para o registro de
  /// dispositivo (FR-017).
  static const String appVersion = '1.0.0';
}
