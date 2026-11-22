import 'dart:async';
import 'dart:io' show Platform;

import 'package:geolocator/geolocator.dart';

import 'leituras_service.dart';

/// Mantém a posição mais recente do celular enquanto o ponto de transmissão
/// está ativo, para carimbar cada leitura (FR-002). A posição fica só em
/// memória — para contas sem empresa nada disso vira histórico (FR-034).
class RastreadorPosicao {
  StreamSubscription<Position>? _assinatura;
  PosicaoLeitura? _ultima;

  static LocationSettings _configuracao() {
    if (Platform.isAndroid) {
      return AndroidSettings(
        accuracy: LocationAccuracy.high,
        distanceFilter: 20,
        intervalDuration: const Duration(seconds: 15),
      );
    }
    if (Platform.isIOS) {
      return AppleSettings(
        accuracy: LocationAccuracy.high,
        distanceFilter: 20,
        activityType: ActivityType.automotiveNavigation,
        pauseLocationUpdatesAutomatically: false,
        allowBackgroundLocationUpdates: true,
        showBackgroundLocationIndicator: true,
      );
    }
    return const LocationSettings(accuracy: LocationAccuracy.high, distanceFilter: 20);
  }

  static PosicaoLeitura _converter(Position p) => PosicaoLeitura(
    latitude: p.latitude,
    longitude: p.longitude,
    precisaoMetros: p.accuracy,
    capturadaEm: DateTime.now(),
  );

  void iniciar() {
    _assinatura ??= Geolocator.getPositionStream(locationSettings: _configuracao()).listen(
      (p) => _ultima = _converter(p),
      onError: (_) {},
    );
  }

  /// Posição com no máximo [LeiturasService.idadeMaximaPosicao]; se a última
  /// estiver velha (parado, sem novas posições), pede uma nova na hora.
  Future<PosicaoLeitura?> obter() async {
    final ultima = _ultima;
    if (ultima != null && DateTime.now().difference(ultima.capturadaEm) <= LeiturasService.idadeMaximaPosicao) {
      return ultima;
    }
    try {
      final p = await Geolocator.getCurrentPosition(
        locationSettings: const LocationSettings(
          accuracy: LocationAccuracy.high,
          timeLimit: Duration(seconds: 10),
        ),
      );
      return _ultima = _converter(p);
    } catch (_) {
      return null;
    }
  }

  Future<void> parar() async {
    await _assinatura?.cancel();
    _assinatura = null;
    _ultima = null;
  }
}
