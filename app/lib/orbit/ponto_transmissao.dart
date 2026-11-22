import 'dart:async';
import 'dart:io' show Platform;

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_reactive_ble/flutter_reactive_ble.dart';
import 'package:hive_flutter/hive_flutter.dart';
import 'package:permission_handler/permission_handler.dart';

import '../core/http_client.dart';
import '../core/token_storage.dart';
import '../data/fila_leituras_local.dart';
import '../services/background_service.dart';
import 'bip.dart';
import 'leitor_orbit.dart';
import 'leituras_service.dart';
import 'orbit_virtual.dart';
import 'rastreador_posicao.dart';

/// Situação da função de ponto de transmissão (data-model.md §6).
enum EstadoPonto { desativado, iniciando, ativo, suspenso }

/// Envio real das leituras: POST /leituras e /ponto/sinal.
class ApiEnviadorLeituras implements EnviadorLeituras {
  ApiEnviadorLeituras({ApiClient? apiClient}) : _api = apiClient ?? ApiClient();

  final ApiClient _api;

  @override
  Future<void> enviarLote(List<Map<String, dynamic>> leituras) async {
    try {
      await _api.dio.post('/leituras', data: {'leituras': leituras});
    } on DioException catch (e) {
      if (e.response?.statusCode == 422) {
        throw LoteRecusado((e.response?.data as Map?)?['mensagem'] as String? ?? 'lote inválido');
      }
      rethrow;
    }
  }

  @override
  Future<void> enviarSinal() => _api.dio.post('/ponto/sinal');
}

/// Liga e desliga o ponto de transmissão (US1) e o modo Orbit virtual (US2).
///
/// - O consentimento fica no servidor (FR-028); a preferência "estava
///   ligado" fica no aparelho, por usuário, para religar sozinho no próximo
///   login.
/// - Sem Bluetooth ou sem permissão, a função fica [EstadoPonto.suspenso]
///   com uma mensagem, sem afetar a rádio.
class PontoTransmissao extends ChangeNotifier {
  PontoTransmissao._();

  static final instance = PontoTransmissao._();

  final _leitor = LeitorOrbit();
  final _rastreador = RastreadorPosicao();
  final _bip = Bip();
  final orbitVirtual = OrbitVirtual();
  late final _api = ApiClient();
  late final leituras = LeiturasService(
    fila: FilaLeiturasLocal(),
    obterPosicao: _rastreador.obter,
    enviador: ApiEnviadorLeituras(apiClient: _api),
    aoRegistrar: () => _bip.tocar(som: bipSom, vibracao: bipVibracao),
  );

  EstadoPonto _estado = EstadoPonto.desativado;
  String? _mensagem;
  StreamSubscription<BleStatus>? _assinaturaEstado;
  Box<dynamic>? _preferencias;

  EstadoPonto get estado => _estado;

  /// Por que a função está suspensa, em linguagem simples.
  String? get mensagem => _mensagem;

  bool get ligado => _estado != EstadoPonto.desativado;

  bool get bipSom => _preferencias?.get('bip_som', defaultValue: true) as bool? ?? true;

  bool get bipVibracao => _preferencias?.get('bip_vibracao', defaultValue: true) as bool? ?? true;

  String get _chaveAtivo => 'ponto_ativo:${TokenStorage.instance.usuario?.id ?? ''}';

  Future<Box<dynamic>> _abrirPreferencias() async =>
      _preferencias ??= await Hive.openBox<dynamic>('preferencias');

  Future<void> definirBip({bool? som, bool? vibracao}) async {
    final prefs = await _abrirPreferencias();
    if (som != null) await prefs.put('bip_som', som);
    if (vibracao != null) await prefs.put('bip_vibracao', vibracao);
    notifyListeners();
  }

  void _definir(EstadoPonto estado, [String? mensagem]) {
    _estado = estado;
    _mensagem = mensagem;
    notifyListeners();
  }

  /// Religa a função depois do login, se o usuário a deixou ligada neste
  /// aparelho. Não pede permissões novas: se faltar alguma, fica suspensa.
  Future<void> retomarSeAtivo() async {
    final prefs = await _abrirPreferencias();
    notifyListeners();
    if (ligado || prefs.get(_chaveAtivo) != true) return;
    await _iniciar(pedirPermissoes: false);
  }

  /// Ativa a função: registra o consentimento no servidor, pede as
  /// permissões e começa a buscar Orbits.
  Future<void> ativar() async {
    final prefs = await _abrirPreferencias();
    _definir(EstadoPonto.iniciando);
    try {
      await _api.dio.put('/me/consentimentos/ponto_transmissao', data: {'aceito': true});
    } catch (_) {
      _definir(EstadoPonto.desativado, 'Não foi possível falar com o servidor. Tente de novo.');
      return;
    }
    await prefs.put(_chaveAtivo, true);
    await _iniciar(pedirPermissoes: true);
  }

  /// Desliga a função e revoga o consentimento (FR-008).
  Future<void> desativar() async {
    final prefs = await _abrirPreferencias();
    await prefs.put(_chaveAtivo, false);
    await _parar();
    _definir(EstadoPonto.desativado);
    try {
      await _api.dio.put('/me/consentimentos/ponto_transmissao', data: {'aceito': false});
    } catch (_) {
      // O servidor já recusaria leituras sem consentimento; sem rede, o
      // aparelho de qualquer forma parou de ler.
    }
  }

  /// Para tudo ao sair da conta, sem revogar o consentimento.
  Future<void> encerrarSessao() async {
    await pararOrbitVirtual();
    await _parar();
    _definir(EstadoPonto.desativado);
  }

  Future<bool> _permissoes({required bool pedir}) async {
    final necessarias = [
      Permission.locationWhenInUse,
      if (Platform.isAndroid) ...[Permission.bluetoothScan, Permission.bluetoothConnect],
      if (Platform.isIOS) Permission.bluetooth,
    ];
    if (pedir) {
      final resultado = await necessarias.request();
      if (resultado.values.any((s) => !s.isGranted && !s.isLimited)) return false;
      if (Platform.isIOS) {
        // Para ler em segundo plano no iPhone, a localização precisa ser
        // "Sempre"; sem ela a função continua valendo com o app aberto.
        await Permission.locationAlways.request();
      }
      return true;
    }
    for (final p in necessarias) {
      if (!(await p.isGranted)) return false;
    }
    return true;
  }

  Future<void> _iniciar({required bool pedirPermissoes}) async {
    _definir(EstadoPonto.iniciando);
    if (!await _permissoes(pedir: pedirPermissoes)) {
      _definir(EstadoPonto.suspenso, 'Permita Bluetooth e localização para o celular ajudar a rede Orbit.');
      return;
    }
    try {
      await BackgroundService.iniciar(MotivoSegundoPlano.pontoTransmissao);
    } catch (_) {
      // Sem o foreground service a função segue com o app aberto.
    }
    _rastreador.iniciar();
    await _assinaturaEstado?.cancel();
    _assinaturaEstado = _leitor.estado.listen(_aoMudarBluetooth);
    _aoMudarBluetooth(_leitor.estadoAtual);
  }

  bool _buscando = false;

  void _aoMudarBluetooth(BleStatus status) {
    switch (status) {
      case BleStatus.ready:
        if (!_buscando) {
          _buscando = true;
          unawaited(leituras.iniciar(_leitor.deteccoes().handleError((_) {
            _buscando = false;
            _definir(EstadoPonto.suspenso, 'A busca por Orbits parou. Toque para tentar de novo.');
          })));
        }
        _definir(EstadoPonto.ativo);
      case BleStatus.poweredOff:
        _pararBusca();
        _definir(EstadoPonto.suspenso, 'Ligue o Bluetooth para o celular ajudar a rede Orbit.');
      case BleStatus.unauthorized:
        _pararBusca();
        _definir(EstadoPonto.suspenso, 'Permita o Bluetooth para o celular ajudar a rede Orbit.');
      case BleStatus.locationServicesDisabled:
        _pararBusca();
        _definir(EstadoPonto.suspenso, 'Ligue a localização (GPS) para o celular ajudar a rede Orbit.');
      case BleStatus.unsupported:
        _pararBusca();
        _definir(EstadoPonto.suspenso, 'Este aparelho não tem Bluetooth LE.');
      case BleStatus.unknown:
        break;
    }
  }

  /// Tenta de novo a partir do estado suspenso (ex.: depois de ligar o
  /// Bluetooth ou conceder a permissão nas configurações).
  Future<void> tentarDeNovo() async {
    await _parar();
    await _iniciar(pedirPermissoes: true);
  }

  void _pararBusca() {
    if (!_buscando) return;
    _buscando = false;
    unawaited(leituras.parar());
  }

  Future<void> _parar() async {
    await _assinaturaEstado?.cancel();
    _assinaturaEstado = null;
    _pararBusca();
    await _rastreador.parar();
    try {
      await BackgroundService.parar(MotivoSegundoPlano.pontoTransmissao);
    } catch (_) {}
  }

  // ---------- Orbit virtual (demonstração) ----------

  /// Papéis que podem usar o modo Orbit virtual (FR-013).
  static bool podeUsarOrbitVirtual(String? papel) => papel == 'admin_ktm' || papel == 'demo';

  Future<void> iniciarOrbitVirtual(String orbitId) async {
    if (!podeUsarOrbitVirtual(TokenStorage.instance.usuario?.papel)) {
      throw StateError('O modo Orbit virtual é só para contas de demonstração.');
    }
    await orbitVirtual.iniciar(orbitId);
    leituras.deduplicador.orbitProprio = orbitId;
    try {
      await BackgroundService.iniciar(MotivoSegundoPlano.orbitVirtual);
    } catch (_) {}
    notifyListeners();
  }

  Future<void> pararOrbitVirtual() async {
    if (orbitVirtual.anunciando.value == null) return;
    await orbitVirtual.parar();
    leituras.deduplicador.orbitProprio = null;
    try {
      await BackgroundService.parar(MotivoSegundoPlano.orbitVirtual);
    } catch (_) {}
    notifyListeners();
  }
}
