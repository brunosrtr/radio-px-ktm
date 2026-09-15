import 'package:flutter/material.dart';

import '../../core/http_client.dart';
import 'canal_ativo_page.dart';

/// Lista de canais autorizados para a empresa do motorista, com opção de
/// silenciar/reativar (FR-019) e alternar entre eles (FR-014/FR-022) — sair
/// de um canal ativo acontece ao voltar da tela `CanalAtivoPage`.
class ListaCanaisPage extends StatefulWidget {
  const ListaCanaisPage({super.key});

  @override
  State<ListaCanaisPage> createState() => _ListaCanaisPageState();
}

class _ListaCanaisPageState extends State<ListaCanaisPage> {
  final _apiClient = ApiClient();

  bool _carregando = true;
  String? _erro;
  List<Map<String, dynamic>> _canais = [];

  @override
  void initState() {
    super.initState();
    _carregar();
  }

  Future<void> _carregar() async {
    setState(() {
      _carregando = true;
      _erro = null;
    });
    try {
      final resposta = await _apiClient.dio.get('/canais');
      final canais = (resposta.data['canais'] as List)
          .cast<Map<String, dynamic>>();
      setState(() => _canais = canais);
    } catch (_) {
      setState(() => _erro = 'Não foi possível carregar os canais.');
    } finally {
      if (mounted) setState(() => _carregando = false);
    }
  }

  Future<void> _alternarSilenciado(Map<String, dynamic> canal) async {
    final novoValor = !(canal['silenciado'] as bool);
    try {
      await _apiClient.dio.put(
        '/canais/${canal['id']}/preferencia',
        data: {'silenciado': novoValor},
      );
      setState(() => canal['silenciado'] = novoValor);
    } catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Não foi possível atualizar a preferência.'),
        ),
      );
    }
  }

  void _entrarNoCanal(Map<String, dynamic> canal) {
    Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => CanalAtivoPage(
          canalId: canal['id'] as String,
          nomeCanal: canal['nome'] as String,
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Canais'),
        actions: [
          IconButton(icon: const Icon(Icons.refresh), onPressed: _carregar),
        ],
      ),
      body: _construirCorpo(),
    );
  }

  Widget _construirCorpo() {
    if (_carregando) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_erro != null) {
      return Center(child: Text(_erro!));
    }
    if (_canais.isEmpty) {
      return const Center(child: Text('Nenhum canal disponível.'));
    }

    return RefreshIndicator(
      onRefresh: _carregar,
      child: ListView.separated(
        itemCount: _canais.length,
        separatorBuilder: (_, _) => const Divider(height: 1),
        itemBuilder: (context, index) {
          final canal = _canais[index];
          final silenciado = canal['silenciado'] as bool;
          return ListTile(
            title: Text(canal['nome'] as String),
            subtitle: Text(
              '${canal['participantes_atual']}/${canal['limite_participantes']} participantes'
              '${(canal['geocerca_ativa'] as bool) ? ' • geocerca ativa' : ''}',
            ),
            leading: IconButton(
              icon: Icon(
                silenciado ? Icons.notifications_off : Icons.notifications_active,
              ),
              tooltip: silenciado ? 'Reativar canal' : 'Silenciar canal',
              onPressed: () => _alternarSilenciado(canal),
            ),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => _entrarNoCanal(canal),
          );
        },
      ),
    );
  }
}
