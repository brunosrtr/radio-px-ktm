import 'package:flutter/material.dart';

import '../../core/http_client.dart';
import '../../core/theme.dart';
import 'canal_ativo_page.dart';
import '../../services/localizacao_service.dart';

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

  final _localizacao = LocalizacaoService();
  String? _avisoLocalizacao;
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
      _avisoLocalizacao = null;
      try {
        await _localizacao.atualizarPosicaoParaEntrada();
      } catch (erro) {
        _avisoLocalizacao = erro is FormatException ? erro.message : 'Não foi possível atualizar o GPS. Canais com área limitada podem não aparecer.';
      }
      final resposta = await _apiClient.dio.get('/canais');
      final canais = (resposta.data['canais'] as List)
          .cast<Map<String, dynamic>>();
      if (!mounted) return;
      setState(() => _canais = canais);
    } catch (_) {
      if (mounted)
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

  Future<void> _entrarNoCanal(Map<String, dynamic> canal) async {
    if (canal['geocerca_ativa'] == true) {
      try {
        await _localizacao.atualizarPosicaoParaEntrada();
      } catch (erro) {
        if (mounted)
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Text(
                erro is FormatException ? erro.message : 'Não foi possível verificar a localização para entrar neste canal.',
              ),
            ),
          );
        return;
      }
    }
    if (!mounted) return;
    await Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => CanalAtivoPage(
          canalId: canal['id'] as String,
          nomeCanal: canal['nome'] as String,
        ),
      ),
    );
    if (mounted) await _carregar();
  }

  String _iniciais(String nome) {
    final partes = nome.trim().split(RegExp(r'\s+'));
    final a = partes.isNotEmpty && partes[0].isNotEmpty ? partes[0][0] : '';
    final b = partes.length > 1 && partes[1].isNotEmpty ? partes[1][0] : '';
    return (a + b).toUpperCase();
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
      body: Column(
        children: [
          if (_avisoLocalizacao != null)
            Padding(
              padding: const EdgeInsets.all(12),
              child: Text(_avisoLocalizacao!),
            ),
          Expanded(child: _construirCorpo()),
        ],
      ),
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
      child: ListView.builder(
        padding: const EdgeInsets.symmetric(vertical: 8),
        itemCount: _canais.length,
        itemBuilder: (context, index) {
          final canal = _canais[index];
          final nome = canal['nome'] as String;
          final silenciado = canal['silenciado'] as bool;
          final geocerca = canal['geocerca_ativa'] as bool;
          final atual = canal['participantes_atual'];
          final limite = canal['limite_participantes'];

          return Padding(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
            child: Material(
              color: Theme.of(context).cardColor,
              borderRadius: BorderRadius.circular(14),
              child: ListTile(
                contentPadding: const EdgeInsets.symmetric(
                  horizontal: 14,
                  vertical: 6,
                ),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(14),
                ),
                leading: CircleAvatar(
                  radius: 24,
                  backgroundColor: AppTheme.corDoAvatar(nome),
                  child: Text(
                    _iniciais(nome),
                    style: const TextStyle(
                      color: Colors.white,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
                title: Text(
                  nome,
                  style: const TextStyle(fontWeight: FontWeight.w600),
                ),
                subtitle: Row(
                  children: [
                    Icon(
                      Icons.group_outlined,
                      size: 14,
                      color: Theme.of(context).textTheme.bodySmall?.color,
                    ),
                    const SizedBox(width: 4),
                    Text('$atual/$limite'),
                    if (geocerca) ...[
                      const SizedBox(width: 10),
                      Icon(
                        Icons.location_on_outlined,
                        size: 14,
                        color: Theme.of(context).textTheme.bodySmall?.color,
                      ),
                      const SizedBox(width: 2),
                      const Text('geocerca'),
                    ],
                  ],
                ),
                trailing: IconButton(
                  icon: Icon(
                    silenciado
                        ? Icons.notifications_off
                        : Icons.notifications_active,
                    color: silenciado ? Colors.grey : AppTheme.azulSecundario,
                  ),
                  tooltip: silenciado ? 'Reativar canal' : 'Silenciar canal',
                  onPressed: () => _alternarSilenciado(canal),
                ),
                onTap: () => _entrarNoCanal(canal),
              ),
            ),
          );
        },
      ),
    );
  }
}
