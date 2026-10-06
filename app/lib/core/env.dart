// Endereços resolvidos na rede local ou sobrescritos por --dart-define.
import 'rede_local.dart';

class Env {
  Env._();

  static String get apiBaseUrl => RedeLocal.instance.apiUrl;
  static String get wsBaseUrl => RedeLocal.instance.wsUrl;

  /// Espelha a versão em pubspec.yaml — enviada no login para o registro de
  /// dispositivo (FR-017).
  static const String appVersion = '1.0.0';
}
