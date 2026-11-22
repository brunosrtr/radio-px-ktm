import 'package:hive_flutter/hive_flutter.dart';

/// Uma leitura de Orbit feita por este celular, ainda não confirmada pelo
/// servidor. Fica só no aparelho de quem leu; o app nunca mostra a
/// identificação do Orbit ao usuário (FR-011).
class LeituraLocal {
  const LeituraLocal({
    required this.orbitId,
    required this.latitude,
    required this.longitude,
    required this.detectadoEm,
    this.precisaoMetros,
    this.rssi,
  });

  final String orbitId;
  final double latitude;
  final double longitude;
  final double? precisaoMetros;
  final int? rssi;
  final DateTime detectadoEm;

  Map<String, dynamic> paraJson() => {
    'orbit_id': orbitId,
    'latitude': latitude,
    'longitude': longitude,
    'precisao_metros': precisaoMetros,
    'rssi': rssi,
    'detectado_em': detectadoEm.toUtc().toIso8601String(),
  };

  static LeituraLocal deMapa(Map<dynamic, dynamic> mapa) => LeituraLocal(
    orbitId: mapa['orbit_id'] as String,
    latitude: (mapa['latitude'] as num).toDouble(),
    longitude: (mapa['longitude'] as num).toDouble(),
    precisaoMetros: (mapa['precisao_metros'] as num?)?.toDouble(),
    rssi: mapa['rssi'] as int?,
    detectadoEm: DateTime.parse(mapa['detectado_em'] as String),
  );
}

/// Contrato da fila offline de leituras, para os testes trocarem o Hive por
/// uma fila em memória.
abstract interface class FilaLeituras {
  Future<void> abrir();
  Future<void> adicionar(LeituraLocal leitura);
  Map<dynamic, LeituraLocal> listarPendentes({int limite});
  Future<void> removerConfirmados(Iterable<dynamic> chaves);
  Future<int> descartarAntigas(DateTime agora);
  int get tamanho;
}

/// Fila offline de leituras (FR-005) sobre Hive. Só remove o que o servidor
/// confirmou e descarta leituras com mais de [idadeMaxima], que o servidor
/// recusaria de qualquer forma.
class FilaLeiturasLocal implements FilaLeituras {
  static const _nomeBox = 'fila_leituras';
  static const idadeMaxima = Duration(days: 7);

  Box<Map>? _box;

  @override
  Future<void> abrir() async {
    _box ??= await Hive.openBox<Map>(_nomeBox);
  }

  @override
  Future<void> adicionar(LeituraLocal leitura) async {
    await _box!.add(leitura.paraJson());
  }

  @override
  Map<dynamic, LeituraLocal> listarPendentes({int limite = 100}) {
    final pendentes = <dynamic, LeituraLocal>{};
    for (final chave in _box!.keys.take(limite)) {
      pendentes[chave] = LeituraLocal.deMapa(_box!.get(chave)!);
    }
    return pendentes;
  }

  @override
  Future<void> removerConfirmados(Iterable<dynamic> chaves) async {
    await _box!.deleteAll(chaves);
  }

  @override
  Future<int> descartarAntigas(DateTime agora) async {
    final limite = agora.subtract(idadeMaxima);
    final antigas = [
      for (final chave in _box!.keys)
        if (DateTime.parse(_box!.get(chave)!['detectado_em'] as String).isBefore(limite)) chave,
    ];
    await _box!.deleteAll(antigas);
    return antigas.length;
  }

  @override
  int get tamanho => _box!.length;
}
