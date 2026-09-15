import 'package:hive_flutter/hive_flutter.dart';

/// Um ponto de localização capturado, ainda não confirmado pelo servidor.
class PontoLocal {
  PontoLocal({
    required this.latitude,
    required this.longitude,
    this.precisaoMetros,
    this.velocidadeKmh,
    required this.capturadoEm,
  });

  final double latitude;
  final double longitude;
  final double? precisaoMetros;
  final double? velocidadeKmh;
  final DateTime capturadoEm;

  Map<String, dynamic> paraJson() => {
    'latitude': latitude,
    'longitude': longitude,
    'precisao_metros': precisaoMetros,
    'velocidade_kmh': velocidadeKmh,
    'capturado_em': capturadoEm.toIso8601String(),
  };

  static PontoLocal deMapa(Map<dynamic, dynamic> mapa) => PontoLocal(
    latitude: mapa['latitude'] as double,
    longitude: mapa['longitude'] as double,
    precisaoMetros: mapa['precisao_metros'] as double?,
    velocidadeKmh: mapa['velocidade_kmh'] as double?,
    capturadoEm: DateTime.parse(mapa['capturado_em'] as String),
  );
}

/// Fila offline de posições (RF-031): grava pontos capturados enquanto o app
/// está sem sinal e só remove os que o servidor efetivamente confirmou —
/// nunca descarta um ponto só porque foi enviado, apenas quando o envio tem
/// sucesso.
class FilaPosicoesLocal {
  static const _nomeBox = 'fila_posicoes';

  Box<Map>? _box;

  Future<void> abrir() async {
    _box ??= await Hive.openBox<Map>(_nomeBox);
  }

  Future<void> adicionar(PontoLocal ponto) async {
    await _box!.add(ponto.paraJson());
  }

  /// Retorna até [limite] pontos pendentes, indexados pela chave interna do
  /// Hive — necessária para remover exatamente esses itens em
  /// [removerConfirmados], mesmo que novos pontos tenham sido adicionados
  /// enquanto o lote estava em trânsito.
  Map<dynamic, PontoLocal> listarPendentes({int limite = 200}) {
    final pendentes = <dynamic, PontoLocal>{};
    for (final chave in _box!.keys.take(limite)) {
      pendentes[chave] = PontoLocal.deMapa(_box!.get(chave)!);
    }
    return pendentes;
  }

  Future<void> removerConfirmados(Iterable<dynamic> chaves) async {
    await _box!.deleteAll(chaves);
  }

  int get tamanho => _box!.length;
}
