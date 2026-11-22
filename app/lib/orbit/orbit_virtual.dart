import 'dart:io' show Platform;

import 'package:flutter/foundation.dart';
import 'package:flutter_ble_peripheral/flutter_ble_peripheral.dart';

import 'orbit_sinal.dart';

/// Modo Orbit virtual (US2): o celular anuncia por Bluetooth o mesmo sinal
/// de um Orbit, para demonstrar a Rede Orbit sem o hardware. Só Android — o
/// iOS não deixa um app anunciar Service Data (research.md §1).
class OrbitVirtual {
  OrbitVirtual({FlutterBlePeripheral? periferico}) : _periferico = periferico ?? FlutterBlePeripheral();

  final FlutterBlePeripheral _periferico;

  /// Orbit anunciado no momento, ou null quando parado.
  final anunciando = ValueNotifier<String?>(null);

  static bool get plataformaSuportada => !kIsWeb && Platform.isAndroid;

  /// Verifica se este aparelho consegue anunciar (alguns Android não têm
  /// suporte a anúncio BLE).
  Future<bool> suportado() async => plataformaSuportada && await _periferico.isSupported;

  /// Começa a anunciar [orbitId]. Lança [StateError] com uma mensagem para o
  /// usuário quando não é possível.
  Future<void> iniciar(String orbitId) async {
    if (!await suportado()) {
      throw StateError('Este aparelho não consegue fazer o papel de Orbit. Use um Android com Bluetooth LE.');
    }
    final permissao = await _periferico.requestPermission();
    if (permissao == PeripheralBluetoothState.denied || permissao == PeripheralBluetoothState.permanentlyDenied) {
      throw StateError('Permita o uso do Bluetooth para anunciar o Orbit virtual.');
    }
    if (permissao == PeripheralBluetoothState.turnedOff) {
      throw StateError('Ligue o Bluetooth para anunciar o Orbit virtual.');
    }
    if (await _periferico.isAdvertising) await _periferico.stop();

    await _periferico.start(
      // Pacote principal: só o Service UUID, que é o que o filtro de busca
      // procura (contracts/sinal-orbit.md).
      advertiseData: const AdvertiseDataCore(serviceUuid: OrbitSinal.uuidRedeOrbit),
      androidSettings: AndroidAdvertiseSettings(
        advertiseSettings: const AdvertiseSettings(
          advertiseMode: AdvertiseMode.advertiseModeLowLatency,
          txPowerLevel: AdvertiseTxPower.advertiseTxPowerHigh,
          connectable: false,
        ),
        // Resposta de varredura: o Service Data com a identificação.
        advertiseResponseData: AndroidAdvertiseData(
          serviceDataUuid: OrbitSinal.uuidRedeOrbit,
          serviceData: OrbitSinal.codificarCarga(orbitId),
        ),
      ),
    );
    anunciando.value = orbitId;
  }

  Future<void> parar() async {
    try {
      await _periferico.stop();
    } finally {
      anunciando.value = null;
    }
  }
}
