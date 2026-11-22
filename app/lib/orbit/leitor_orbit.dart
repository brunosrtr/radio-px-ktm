import 'dart:async';

import 'package:flutter_reactive_ble/flutter_reactive_ble.dart';

import 'orbit_sinal.dart';

/// Um sinal de Orbit captado pelo rádio Bluetooth deste celular.
class DeteccaoOrbit {
  const DeteccaoOrbit({required this.orbitId, required this.rssi, required this.instante});

  final String orbitId;
  final int rssi;
  final DateTime instante;
}

/// Busca contínua por sinais de Orbit (FR-001), sempre filtrada pelo
/// [OrbitSinal.uuidRedeOrbit] — exigência do iOS para entregar anúncios em
/// segundo plano e do Android para não restringir buscas sem filtro com a
/// tela apagada. O modo `balanced` deixa o próprio sistema alternar entre
/// escutar e descansar, para economizar bateria (SC-007).
class LeitorOrbit {
  LeitorOrbit({FlutterReactiveBle? ble}) : _ble = ble ?? FlutterReactiveBle();

  final FlutterReactiveBle _ble;
  final _uuid = Uuid.parse(OrbitSinal.uuidRedeOrbit);

  /// Estado do Bluetooth do aparelho (desligado, sem permissão, pronto...).
  Stream<BleStatus> get estado => _ble.statusStream;

  BleStatus get estadoAtual => _ble.status;

  /// Detecções de Orbit enquanto houver alguém escutando. Cancelar a
  /// assinatura encerra a busca.
  Stream<DeteccaoOrbit> deteccoes({ScanMode modo = ScanMode.balanced}) {
    return _ble
        .scanForDevices(withServices: [_uuid], scanMode: modo)
        .map((dispositivo) {
          final orbitId = OrbitSinal.lerCarga(dispositivo.serviceData[_uuid]);
          if (orbitId == null) return null;
          return DeteccaoOrbit(orbitId: orbitId, rssi: dispositivo.rssi, instante: DateTime.now());
        })
        .where((d) => d != null)
        .cast<DeteccaoOrbit>();
  }
}
