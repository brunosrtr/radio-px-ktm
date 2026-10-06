import 'dart:async';
import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:hive/hive.dart';
import 'package:nsd/nsd.dart' as nsd;

class ServidorLocal {
  const ServidorLocal({
    required this.id,
    required this.nome,
    required this.url,
  });
  final String id;
  final String nome;
  final String url;

  static String validarUrl(String valor) {
    final uri = Uri.tryParse(valor);
    if (uri == null ||
        !['http', 'https'].contains(uri.scheme) ||
        uri.host.isEmpty ||
        uri.userInfo.isNotEmpty ||
        uri.hasQuery ||
        uri.hasFragment ||
        (uri.path.isNotEmpty && uri.path != '/') ||
        uri.port < 1 ||
        uri.port > 65535) {
      throw const FormatException('Endereço de computador inválido.');
    }
    return uri.replace(path: '').toString();
  }

  factory ServidorLocal.doQr(String texto) {
    final dados = jsonDecode(texto);
    if (dados is! Map ||
        dados['service'] != 'radio-px-ktm' ||
        dados['version'] != 1 ||
        dados['id'] is! String ||
        (dados['id'] as String).isEmpty ||
        dados['url'] is! String) {
      throw const FormatException('Este QR Code não é do Rádio PX.');
    }
    return ServidorLocal(
      id: dados['id'],
      nome: 'Rádio PX KTM',
      url: validarUrl(dados['url']),
    );
  }
}

/// A descoberta informa endereços, não substitui o login ou TLS.
/// Nunca envia credenciais ao consultar /local/info.
class RedeLocal extends ChangeNotifier {
  RedeLocal._();
  @visibleForTesting
  RedeLocal.paraTeste({
    required Future<List<String>> Function() descobrir,
    required Future<ServidorLocal?> Function(String, String?) verificar,
  }) : _descobrirTeste = descobrir,
       _verificarTeste = verificar;
  Future<List<String>> Function()? _descobrirTeste;
  Future<ServidorLocal?> Function(String, String?)? _verificarTeste;
  static final instance = RedeLocal._();
  static const urlFixa = String.fromEnvironment('API_BASE_URL');
  static const wsFixo = String.fromEnvironment('WS_BASE_URL');
  static const temUrlFixa = urlFixa != '';

  Box<dynamic>? _box;
  ServidorLocal? servidor;
  List<ServidorLocal> encontrados = [];
  bool procurando = false;
  String? erro;
  Future<void>? _busca;
  DateTime? _validadoEm;
  bool get inicializada => _box != null;
  String get apiUrl => temUrlFixa ? urlFixa : servidor?.url ?? '';
  String get wsUrl => temUrlFixa && wsFixo.isNotEmpty
      ? wsFixo
      : apiUrl.replaceFirst(RegExp(r'^http'), 'ws');

  Future<void> iniciar() async {
    if (temUrlFixa) return;
    _box = await Hive.openBox('servidor_local');
    final id = _box!.get('id');
    final url = _box!.get('url');
    if (id is String && url is String) {
      try {
        servidor = ServidorLocal(
          id: id,
          nome: _box!.get('nome', defaultValue: 'Rádio PX KTM'),
          url: ServidorLocal.validarUrl(url),
        );
      } catch (_) {
        await _box!.clear();
      }
    }
  }

  Future<ServidorLocal?> _verificar(String url, {String? id}) async {
    if (_verificarTeste != null) return _verificarTeste!(url, id);
    final dio = Dio(
      BaseOptions(
        connectTimeout: const Duration(seconds: 2),
        receiveTimeout: const Duration(seconds: 2),
        followRedirects: false,
      ),
    );
    try {
      final resposta = await dio.get('$url/local/info');
      final dados = resposta.data;
      if (dados is! Map ||
          dados['service'] != 'radio-px-ktm' ||
          dados['version'] != 1 ||
          dados['id'] is! String ||
          (dados['id'] as String).isEmpty ||
          (id != null && dados['id'] != id)) {
        return null;
      }
      return ServidorLocal(
        id: dados['id'],
        nome: dados['name'] is String ? dados['name'] : 'Rádio PX KTM',
        url: url,
      );
    } catch (_) {
      return null;
    } finally {
      dio.close(force: true);
    }
  }

  Future<void> _salvar(ServidorLocal atual) async {
    servidor = atual;
    _validadoEm = DateTime.now();
    await _box?.putAll({'id': atual.id, 'nome': atual.nome, 'url': atual.url});
    erro = null;
    notifyListeners();
  }

  Future<void> selecionar(ServidorLocal escolhido) async {
    final validado = await _verificar(escolhido.url, id: escolhido.id);
    if (validado == null) {
      throw const FormatException(
        'Computador inacessível. Confira se ambos estão no mesmo Wi-Fi.',
      );
    }
    await _salvar(validado);
  }

  Future<void> lerQr(String texto) => selecionar(ServidorLocal.doQr(texto));

  Future<void> assegurarConexao({bool forcar = false}) async {
    if (temUrlFixa) return;
    if (!forcar &&
        _validadoEm != null &&
        DateTime.now().difference(_validadoEm!) < const Duration(seconds: 10)) {
      return;
    }
    await procurar();
    if (_validadoEm == null) {
      throw StateError(erro ?? 'Selecione o computador antes de entrar.');
    }
  }

  Future<void> procurar() {
    if (temUrlFixa) return Future.value();
    return _busca ??= _procurar().whenComplete(() => _busca = null);
  }

  Future<void> _procurar() async {
    procurando = true;
    erro = null;
    _validadoEm = null;
    notifyListeners();
    try {
      if (servidor != null) {
        final atual = await _verificar(servidor!.url, id: servidor!.id);
        if (atual != null) {
          await _salvar(atual);
          return;
        }
      }
      encontrados = [];
      final candidatos =
          await (_descobrirTeste?.call() ?? _descobrirEnderecos());
      final resultados = await Future.wait(
        candidatos.take(16).map((url) => _verificar(url)),
      );
      final porId = <String, ServidorLocal>{};
      for (final atual in resultados.whereType<ServidorLocal>()) {
        porId.putIfAbsent(atual.id, () => atual);
      }
      encontrados = porId.values.toList();
      final mesmo = encontrados.where((s) => s.id == servidor?.id);
      if (servidor != null && mesmo.isNotEmpty) {
        await _salvar(mesmo.first);
      } else if (servidor == null && encontrados.length == 1) {
        await _salvar(encontrados.single);
      } else {
        erro = servidor != null
            ? 'Computador salvo não encontrado. Confira o Wi-Fi ou leia o QR Code no painel.'
            : encontrados.isEmpty
            ? 'Nenhum computador encontrado. Confira o Wi-Fi ou leia o QR Code no painel.'
            : 'Escolha o computador da sua empresa.';
      }
    } catch (_) {
      erro = 'Não foi possível procurar na rede. Permita o acesso à rede local ou leia o QR Code.';
    } finally {
      procurando = false;
      notifyListeners();
    }
  }

  Future<List<String>> _descobrirEnderecos() async {
    nsd.Discovery? descoberta;
    try {
      descoberta = await nsd.startDiscovery(
        '_radiopx._tcp',
        ipLookupType: nsd.IpLookupType.v4,
      );
      // Janela limitada: libera o multicast ao terminar, inclusive em erro.
      await Future<void>.delayed(const Duration(seconds: 4));
      final candidatos = <String>{};
      for (final servico in descoberta.services) {
        final porta = servico.port;
        if (porta == null || porta < 1 || porta > 65535) continue;
        for (final ip in servico.addresses ?? []) {
          candidatos.add(
            Uri(scheme: 'http', host: ip.address, port: porta).toString(),
          );
        }
        if ((servico.addresses?.isEmpty ?? true) && servico.host != null) {
          candidatos.add(
            Uri(scheme: 'http', host: servico.host, port: porta).toString(),
          );
        }
      }

      return candidatos.toList();
    } finally {
      if (descoberta != null) {
        try {
          await nsd.stopDiscovery(descoberta);
        } catch (_) {}
      }
    }
  }
}
