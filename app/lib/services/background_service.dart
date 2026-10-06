import 'package:flutter/foundation.dart'
    show kIsWeb, defaultTargetPlatform, TargetPlatform;
import 'package:permission_handler/permission_handler.dart';
import 'package:flutter_foreground_task/flutter_foreground_task.dart';

/// Mantém o processo do app vivo em segundo plano no Android via um
/// foreground service com notificação persistente — o Android não mata o
/// processo enquanto ela estiver visível, então a conexão WebSocket
/// (`canal_service.dart`) e a coleta de localização
/// (`localizacao_service.dart`) continuam rodando normalmente na isolate
/// principal do app (RF-07, RNF-09).
///
/// O `TaskHandler` abaixo não reimplementa a lógica de voz/localização numa
/// isolate separada — seu único papel é sustentar essa notificação. Em iOS
/// não existe um foreground service equivalente; o app depende dos
/// "background modes" nativos (location) declarados em
/// `ios/Runner/Info.plist`.
class BackgroundService {
  BackgroundService._();

  static bool _inicializado = false;

  static void _inicializar() {
    if (_inicializado) return;
    _inicializado = true;

    FlutterForegroundTask.init(
      androidNotificationOptions: AndroidNotificationOptions(
        channelId: 'radio_px_canal_ativo',
        channelName: 'Rádio PX — canal ativo',
        channelDescription:
            'Mantém a conexão de voz e a localização ativas em segundo plano.',
      ),
      iosNotificationOptions: const IOSNotificationOptions(),
      foregroundTaskOptions: ForegroundTaskOptions(
        eventAction: ForegroundTaskEventAction.repeat(60000),
        autoRunOnBoot: false,
        allowWifiLock: true,
      ),
    );
  }

  /// Inicia o foreground service, se ainda não estiver rodando. No-op no
  /// Flutter Web: não existe processo em segundo plano fora da aba do
  /// navegador, então não há o que sustentar.
  static Future<void> iniciar() async {
    if (kIsWeb) return;
    _inicializar();
    if (await FlutterForegroundTask.isRunningService) return;
    if (defaultTargetPlatform == TargetPlatform.android) {
      final permissions = await [
        Permission.microphone,
        Permission.locationWhenInUse,
      ].request();
      if (permissions.values.any((status) => !status.isGranted)) {
        throw StateError(
          'Permita microfone e localização para entrar no canal.',
        );
      }
      await FlutterForegroundTask.requestNotificationPermission();
    }

    final result = await FlutterForegroundTask.startService(
      notificationTitle: 'Rádio PX Digital',
      notificationText: 'Canal de voz e localização ativos',
      callback: iniciarTarefaDeSegundoPlano,
    );
    if (result is ServiceRequestFailure) throw result.error;
  }

  static Future<void> parar() {
    if (kIsWeb) return Future.value();
    return FlutterForegroundTask.stopService();
  }
}

@pragma('vm:entry-point')
void iniciarTarefaDeSegundoPlano() {
  FlutterForegroundTask.setTaskHandler(_TarefaMantemNotificacaoViva());
}

class _TarefaMantemNotificacaoViva extends TaskHandler {
  @override
  Future<void> onStart(DateTime timestamp, TaskStarter starter) async {}

  @override
  void onRepeatEvent(DateTime timestamp) {}

  @override
  Future<void> onDestroy(DateTime timestamp, bool isTimeout) async {}
}
