import 'package:flutter/foundation.dart'
    show kIsWeb, defaultTargetPlatform, TargetPlatform;
import 'package:permission_handler/permission_handler.dart';
import 'package:flutter_foreground_task/flutter_foreground_task.dart';

/// Por que o app precisa continuar rodando em segundo plano. Cada motivo
/// pede tipos diferentes de foreground service ao Android 14+.
enum MotivoSegundoPlano {
  /// Canal de voz aberto: microfone, reprodução e localização (geocerca).
  canalDeVoz,

  /// Ponto de transmissão: busca Bluetooth por Orbits + localização da
  /// leitura (003-rede-orbit, FR-006).
  pontoTransmissao,

  /// Modo Orbit virtual: anúncio Bluetooth (demonstração).
  orbitVirtual,
}

/// Mantém o processo do app vivo em segundo plano no Android via um
/// foreground service com notificação persistente — o Android não mata o
/// processo enquanto ela estiver visível, então a conexão WebSocket
/// (`canal_service.dart`), a coleta de localização
/// (`localizacao_service.dart`) e a busca por Orbits (`orbit/`) continuam
/// rodando normalmente na isolate principal do app (RF-07, RNF-09).
///
/// O serviço é compartilhado: cada recurso que precisa dele se registra com
/// um [MotivoSegundoPlano], e o serviço só para quando o último motivo sai.
///
/// O `TaskHandler` abaixo não reimplementa nenhuma lógica numa isolate
/// separada — seu único papel é sustentar essa notificação. Em iOS não existe
/// um foreground service equivalente; o app depende dos "background modes"
/// nativos declarados em `ios/Runner/Info.plist`.
class BackgroundService {
  BackgroundService._();

  static bool _inicializado = false;
  static final _motivos = <MotivoSegundoPlano>{};
  static Set<MotivoSegundoPlano> _motivosNoServico = {};

  static void _inicializar() {
    if (_inicializado) return;
    _inicializado = true;

    FlutterForegroundTask.init(
      androidNotificationOptions: AndroidNotificationOptions(
        channelId: 'radio_px_canal_ativo',
        channelName: 'Rádio PX — em segundo plano',
        channelDescription:
            'Mantém o canal de voz e o ponto de transmissão ativos em segundo plano.',
      ),
      iosNotificationOptions: const IOSNotificationOptions(),
      foregroundTaskOptions: ForegroundTaskOptions(
        eventAction: ForegroundTaskEventAction.repeat(60000),
        autoRunOnBoot: false,
        allowWifiLock: true,
      ),
    );
  }

  static List<ForegroundServiceTypes> _tipos(Set<MotivoSegundoPlano> motivos) => {
    for (final motivo in motivos)
      ...switch (motivo) {
        MotivoSegundoPlano.canalDeVoz => [
          ForegroundServiceTypes.microphone,
          ForegroundServiceTypes.mediaPlayback,
          ForegroundServiceTypes.location,
        ],
        MotivoSegundoPlano.pontoTransmissao => [
          ForegroundServiceTypes.location,
          ForegroundServiceTypes.connectedDevice,
        ],
        MotivoSegundoPlano.orbitVirtual => [ForegroundServiceTypes.connectedDevice],
      },
  }.toList();

  static String _texto(Set<MotivoSegundoPlano> motivos) {
    final partes = [
      if (motivos.contains(MotivoSegundoPlano.canalDeVoz)) 'canal de voz',
      if (motivos.contains(MotivoSegundoPlano.pontoTransmissao)) 'ponto de transmissão',
      if (motivos.contains(MotivoSegundoPlano.orbitVirtual)) 'Orbit virtual (demonstração)',
    ];
    final lista = partes.length > 1
        ? '${partes.sublist(0, partes.length - 1).join(', ')} e ${partes.last}'
        : partes.join();
    return '${lista[0].toUpperCase()}${lista.substring(1)} ativo${partes.length > 1 ? 's' : ''}';
  }

  /// Registra [motivo] e garante o foreground service rodando com os tipos
  /// necessários. No-op no Flutter Web e no iOS (sem foreground service).
  static Future<void> iniciar([
    MotivoSegundoPlano motivo = MotivoSegundoPlano.canalDeVoz,
  ]) async {
    if (kIsWeb) return;
    if (defaultTargetPlatform != TargetPlatform.android) {
      _motivos.add(motivo);
      return;
    }
    _inicializar();

    if (motivo == MotivoSegundoPlano.canalDeVoz) {
      final permissions = await [
        Permission.microphone,
        Permission.locationWhenInUse,
      ].request();
      if (permissions.values.any((status) => !status.isGranted)) {
        throw StateError(
          'Permita microfone e localização para entrar no canal.',
        );
      }
    }
    await FlutterForegroundTask.requestNotificationPermission();

    _motivos.add(motivo);
    await _aplicar();
  }

  /// Retira [motivo]; o serviço para quando não sobra nenhum.
  static Future<void> parar([
    MotivoSegundoPlano motivo = MotivoSegundoPlano.canalDeVoz,
  ]) async {
    if (kIsWeb) return;
    _motivos.remove(motivo);
    if (defaultTargetPlatform != TargetPlatform.android) return;
    await _aplicar();
  }

  static Future<void> _aplicar() async {
    final rodando = await FlutterForegroundTask.isRunningService;
    if (_motivos.isEmpty) {
      if (rodando) await FlutterForegroundTask.stopService();
      _motivosNoServico = {};
      return;
    }
    if (rodando && _mesmos(_motivosNoServico, _motivos)) return;

    // Os tipos do foreground service só valem no início; quando os motivos
    // mudam, o serviço é reiniciado com o novo conjunto.
    if (rodando) await FlutterForegroundTask.stopService();
    final motivos = {..._motivos};
    final result = await FlutterForegroundTask.startService(
      serviceTypes: _tipos(motivos),
      notificationTitle: 'Rádio PX KTM',
      notificationText: _texto(motivos),
      callback: iniciarTarefaDeSegundoPlano,
    );
    if (result is ServiceRequestFailure) throw result.error;
    _motivosNoServico = motivos;
  }
}

bool _mesmos<T>(Set<T> a, Set<T> b) => a.length == b.length && a.containsAll(b);

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
