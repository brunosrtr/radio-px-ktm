import 'dart:async';

import 'package:geolocator/geolocator.dart';

import '../core/http_client.dart';
import '../data/fila_posicoes_local.dart';

/// Decide se um novo ponto deve ser persistido, segundo o filtro de
/// deslocamento/intervalo (RF-024, RF-028): um novo ponto a cada 200m ou
/// 30s, o que ocorrer primeiro. Extraída como função pura para ser testável
/// sem depender do `geolocator` (app/test/localizacao_service_test.dart).
bool devePersistirPonto({
  required double distanciaMetros,
  required Duration duracaoDesdeUltimoPonto,
}) {
  return distanciaMetros >= LocalizacaoService.distanciaMinimaMetros ||
      duracaoDesdeUltimoPonto >= LocalizacaoService.intervaloMinimo;
}

/// Coleta localização em segundo plano, grava numa fila offline local e
/// envia em lote periodicamente, removendo do dispositivo apenas o que o
/// servidor confirmou (RF-024, RF-025, RF-031, RNF-015).
class LocalizacaoService {
  LocalizacaoService({ApiClient? apiClient, FilaPosicoesLocal? fila})
    : _apiClient = apiClient ?? ApiClient(),
      _fila = fila ?? FilaPosicoesLocal();

  static const distanciaMinimaMetros = 200.0;
  static const intervaloMinimo = Duration(seconds: 30);
  static const intervaloEnvio = Duration(minutes: 1, seconds: 30);

  final ApiClient _apiClient;
  final FilaPosicoesLocal _fila;

  StreamSubscription<Position>? _assinaturaPosicoes;
  Timer? _timerEnvio;

  Position? _ultimaCapturada;
  DateTime? _ultimaCapturadaEm;

  Future<void> iniciar() async {
    await _fila.abrir();

    var permissao = await Geolocator.checkPermission();
    if (permissao == LocationPermission.denied) {
      permissao = await Geolocator.requestPermission();
    }
    if (permissao == LocationPermission.denied ||
        permissao == LocationPermission.deniedForever) {
      return;
    }

    _assinaturaPosicoes = Geolocator.getPositionStream(
      locationSettings: const LocationSettings(
        accuracy: LocationAccuracy.high,
      ),
    ).listen(_processarNovaPosicao);

    _timerEnvio = Timer.periodic(intervaloEnvio, (_) => enviarLotePendente());
  }

  void _processarNovaPosicao(Position posicao) {
    final agora = DateTime.now();

    if (!_deveRegistrar(posicao, agora)) return;

    _ultimaCapturada = posicao;
    _ultimaCapturadaEm = agora;

    unawaited(
      _fila.adicionar(
        PontoLocal(
          latitude: posicao.latitude,
          longitude: posicao.longitude,
          precisaoMetros: posicao.accuracy,
          velocidadeKmh: posicao.speed * 3.6,
          capturadoEm: agora.toUtc(),
        ),
      ),
    );
  }

  bool _deveRegistrar(Position posicao, DateTime agora) {
    if (_ultimaCapturada == null || _ultimaCapturadaEm == null) return true;

    final distancia = Geolocator.distanceBetween(
      _ultimaCapturada!.latitude,
      _ultimaCapturada!.longitude,
      posicao.latitude,
      posicao.longitude,
    );

    return devePersistirPonto(
      distanciaMetros: distancia,
      duracaoDesdeUltimoPonto: agora.difference(_ultimaCapturadaEm!),
    );
  }

  /// Envia os pontos pendentes em lote e remove da fila local apenas os que
  /// o servidor confirmou — em caso de falha (sem sinal, erro do servidor),
  /// os pontos continuam na fila para o próximo ciclo.
  Future<void> enviarLotePendente() async {
    final pendentes = _fila.listarPendentes();
    if (pendentes.isEmpty) return;

    try {
      await _apiClient.dio.post(
        '/posicoes',
        data: {'pontos': pendentes.values.map((p) => p.paraJson()).toList()},
      );
      await _fila.removerConfirmados(pendentes.keys);
    } catch (_) {
      // Mantém os pontos na fila; serão reenviados no próximo ciclo.
    }
  }

  Future<void> parar() async {
    await _assinaturaPosicoes?.cancel();
    _timerEnvio?.cancel();
  }
}
