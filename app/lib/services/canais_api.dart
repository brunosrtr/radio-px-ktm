import 'package:dio/dio.dart';

import '../core/http_client.dart';

/// Erro das rotas de canais, com o `codigo` do contrato e uma mensagem
/// pronta para o usuário.
class ErroCanal implements Exception {
  const ErroCanal(this.codigo, this.mensagem);

  final String codigo;
  final String mensagem;

  @override
  String toString() => mensagem;
}

/// Canal como as listas e telas do app usam.
class Canal {
  const Canal({
    required this.id,
    required this.nome,
    this.descricao = '',
    this.limite = 10,
    this.participantes = 0,
    this.geocerca = false,
    this.silenciado = false,
    this.favorito = false,
    this.souCriador = false,
    this.podeGerenciar = false,
    this.comunidade = true,
    this.publico = true,
    this.codigoConvite,
    this.distanciaMetros,
    this.raioMetros,
  });

  factory Canal.deJson(Map<String, dynamic> j) => Canal(
    id: j['id'] as String,
    nome: j['nome'] as String,
    descricao: (j['descricao'] as String?) ?? '',
    limite: (j['limite_participantes'] as int?) ?? 10,
    participantes: (j['participantes_atual'] as int?) ?? 0,
    geocerca: (j['geocerca_ativa'] as bool?) ?? false,
    silenciado: (j['silenciado'] as bool?) ?? false,
    favorito: (j['favorito'] as bool?) ?? false,
    souCriador: (j['sou_criador'] as bool?) ?? false,
    podeGerenciar: (j['pode_gerenciar'] as bool?) ?? false,
    comunidade: (j['comunidade'] as bool?) ?? true,
    publico: (j['visibilidade'] as String?) != 'privado',
    codigoConvite: j['codigo_convite'] as String?,
    distanciaMetros: (j['distancia_metros'] as num?)?.toDouble(),
    raioMetros: j['raio_metros'] as int?,
  );

  final String id;
  final String nome;
  final String descricao;
  final int limite;
  final int participantes;
  final bool geocerca;
  final bool silenciado;
  final bool favorito;
  final bool souCriador;
  final bool podeGerenciar;
  final bool comunidade;
  final bool publico;
  final String? codigoConvite;
  final double? distanciaMetros;
  final int? raioMetros;

  bool get privado => comunidade && !publico;
}

/// Participante conectado agora num canal (para o criador remover).
class Participante {
  const Participante(this.usuarioId, this.nome);

  final String usuarioId;
  final String nome;
}

/// Monta o link de convite que vai no QR Code.
String linkConvite(String codigo) => 'radiopx://app/convite/${codigo.replaceAll('-', '')}';

/// Aceita o que a pessoa digita ou o link lido no QR e devolve o código no
/// formato XXXX-XXXX, ou null se não for um código válido — mesma regra do
/// backend (`canal.NormalizarCodigoConvite`).
String? normalizarCodigoConvite(String entrada) {
  var texto = entrada.trim();
  final barra = texto.lastIndexOf('/');
  if (barra >= 0) texto = texto.substring(barra + 1);
  texto = texto.toUpperCase().replaceAll(RegExp(r'[-\s]'), '');
  if (!RegExp(r'^[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{8}$').hasMatch(texto)) return null;
  return '${texto.substring(0, 4)}-${texto.substring(4)}';
}

/// Cliente das rotas de canais da comunidade (contracts/rest-api.md).
class CanaisApi {
  CanaisApi({ApiClient? apiClient}) : _api = apiClient ?? ApiClient();

  final ApiClient _api;

  Future<T> _executar<T>(Future<T> Function() chamada) async {
    try {
      return await chamada();
    } on DioException catch (e) {
      final dados = e.response?.data;
      if (dados is Map && dados['codigo'] is String) {
        final codigo = dados['codigo'] as String;
        throw ErroCanal(codigo, _mensagem(codigo, dados['mensagem'] as String?));
      }
      throw const ErroCanal('sem_conexao', 'Não foi possível falar com o servidor. Confira sua conexão.');
    }
  }

  static String _mensagem(String codigo, String? servidor) => switch (codigo) {
    'convite_invalido' => 'Convite inválido ou trocado pelo criador. Peça um novo.',
    'banido' => 'O criador removeu você deste canal.',
    'sem_permissao' => 'Você não tem acesso a este canal.',
    'nao_encontrado' => 'Este canal não existe mais.',
    'fora_geocerca' => 'Você está fora da área deste canal.',
    _ => servidor ?? 'Algo deu errado. Tente de novo.',
  };

  Future<List<Canal>> listar() => _executar(() async {
    final r = await _api.dio.get('/canais');
    return [for (final c in r.data['canais'] as List) Canal.deJson((c as Map).cast<String, dynamic>())];
  });

  Future<List<Canal>> explorar({String busca = '', double? lat, double? lng, bool perto = false}) => _executar(() async {
    final r = await _api.dio.get(
      '/canais/explorar',
      queryParameters: {
        if (busca.isNotEmpty) 'q': busca,
        'lat': ?lat,
        'lng': ?lng,
        if (perto) 'perto': 'true',
      },
    );
    return [for (final c in r.data['canais'] as List) Canal.deJson((c as Map).cast<String, dynamic>())];
  });

  Future<Canal> detalhar(String id) => _executar(() async {
    final r = await _api.dio.get('/canais/$id');
    return Canal.deJson((r.data as Map).cast<String, dynamic>());
  });

  Future<Canal> criar({
    required String nome,
    required String descricao,
    required bool publico,
    required int limite,
    double? latitude,
    double? longitude,
    int? raioMetros,
  }) => _executar(() async {
    final r = await _api.dio.post(
      '/canais',
      data: {
        'nome': nome,
        'descricao': descricao,
        'visibilidade': publico ? 'publico' : 'privado',
        'limite_participantes': limite,
        if (latitude != null && longitude != null && raioMetros != null)
          'geocerca': {'latitude': latitude, 'longitude': longitude, 'raio_metros': raioMetros},
      },
    );
    return Canal.deJson({...(r.data as Map).cast<String, dynamic>(), 'sou_criador': true, 'pode_gerenciar': true});
  });

  Future<void> apagar(String id) => _executar(() => _api.dio.delete('/canais/$id'));

  Future<String> novoConvite(String id) => _executar(() async {
    final r = await _api.dio.post('/canais/$id/convite');
    return r.data['codigo_convite'] as String;
  });

  /// Entra num canal privado pelo código; devolve (id, nome) do canal.
  Future<Canal> entrarPorConvite(String codigo) => _executar(() async {
    final r = await _api.dio.post('/convites/$codigo');
    return Canal.deJson((r.data as Map).cast<String, dynamic>());
  });

  Future<List<Participante>> participantes(String id) => _executar(() async {
    final r = await _api.dio.get('/canais/$id/participantes');
    return [
      for (final p in r.data['participantes'] as List)
        Participante((p as Map)['usuario_id'] as String, p['nome'] as String),
    ];
  });

  Future<void> remover(String canalId, String usuarioId) =>
      _executar(() => _api.dio.delete('/canais/$canalId/participantes/$usuarioId'));

  Future<void> favoritar(String id, bool favorito) =>
      _executar(() => _api.dio.put('/canais/$id/favorito', data: {'favorito': favorito}));
}
