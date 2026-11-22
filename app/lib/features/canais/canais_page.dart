import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/http_client.dart';
import '../../core/router.dart';
import '../../core/theme.dart';
import '../../core/token_storage.dart';
import '../../services/localizacao_service.dart';
import '../ponto/cartao_ponto_transmissao.dart';

/// Aba Canais: os canais que a pessoa pode usar, com o ponto de transmissão
/// em destaque. Silenciar fica dentro do canal (FR-019); a lista mostra quem
/// está silenciado.
class CanaisPage extends StatefulWidget {
  const CanaisPage({super.key});

  @override
  State<CanaisPage> createState() => _CanaisPageState();
}

class _CanaisPageState extends State<CanaisPage> {
  final _apiClient = ApiClient();
  final _localizacao = LocalizacaoService();
  String? _avisoLocalizacao;
  bool _carregando = true;
  String? _erro;
  List<Map<String, dynamic>> _canais = [];

  bool get _temEmpresa => TokenStorage.instance.usuario?.temEmpresa ?? false;

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
        _avisoLocalizacao = erro is FormatException
            ? erro.message
            : 'Não foi possível atualizar o GPS. Canais com área limitada podem não aparecer.';
      }
      final resposta = await _apiClient.dio.get('/canais');
      final canais = (resposta.data['canais'] as List).cast<Map<String, dynamic>>();
      if (mounted) setState(() => _canais = canais);
    } catch (_) {
      if (mounted) setState(() => _erro = 'Não foi possível carregar os canais. Puxe para tentar de novo.');
    } finally {
      if (mounted) setState(() => _carregando = false);
    }
  }

  Future<void> _entrar(Map<String, dynamic> canal) async {
    if (canal['geocerca_ativa'] == true) {
      try {
        await _localizacao.atualizarPosicaoParaEntrada();
      } catch (erro) {
        if (!mounted) return;
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
    await context.push(
      '/canal/${canal['id']}',
      extra: AberturaCanal(nome: canal['nome'] as String, silenciado: canal['silenciado'] as bool? ?? false),
    );
    if (mounted) await _carregar();
  }

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    return SafeArea(
      bottom: false,
      child: RefreshIndicator(
        onRefresh: _carregar,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(4, 8, 4, 12),
              child: Text('Canais', style: tema.textTheme.headlineMedium?.copyWith(fontWeight: FontWeight.w700)),
            ),
            const CartaoPontoTransmissao(),
            const SizedBox(height: 16),
            if (_avisoLocalizacao != null)
              Padding(
                padding: const EdgeInsets.only(bottom: 12),
                child: Text(_avisoLocalizacao!, style: tema.textTheme.bodyMedium?.copyWith(color: tema.colorScheme.onSurfaceVariant)),
              ),
            if (_carregando)
              const Padding(padding: EdgeInsets.all(32), child: Center(child: CircularProgressIndicator()))
            else if (_erro != null)
              Padding(padding: const EdgeInsets.all(24), child: Text(_erro!, textAlign: TextAlign.center))
            else if (_canais.isEmpty)
              _Vazio(temEmpresa: _temEmpresa)
            else ...[
              Padding(
                padding: const EdgeInsets.fromLTRB(4, 0, 4, 8),
                child: Text(
                  'SEUS CANAIS',
                  style: tema.textTheme.labelMedium?.copyWith(
                    letterSpacing: 1.2,
                    fontWeight: FontWeight.w700,
                    color: tema.colorScheme.onSurfaceVariant,
                  ),
                ),
              ),
              for (final canal in _canais) ...[
                _ItemCanal(canal: canal, aoTocar: () => _entrar(canal)),
                const SizedBox(height: 8),
              ],
            ],
          ],
        ),
      ),
    );
  }
}

class _ItemCanal extends StatelessWidget {
  const _ItemCanal({required this.canal, required this.aoTocar});

  final Map<String, dynamic> canal;
  final VoidCallback aoTocar;

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    final nome = canal['nome'] as String;
    final silenciado = canal['silenciado'] as bool? ?? false;
    final geocerca = canal['geocerca_ativa'] as bool? ?? false;
    final detalhes = [
      '${canal['participantes_atual']} de ${canal['limite_participantes']}',
      if (geocerca) 'área limitada',
      if (silenciado) 'silenciado',
    ].join(' · ');
    return Card(
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: aoTocar,
        child: ConstrainedBox(
          constraints: const BoxConstraints(minHeight: 72),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
            child: Row(
              children: [
                CircleAvatar(
                  radius: 24,
                  backgroundColor: AppTheme.corDoAvatar(nome),
                  child: Text(AppTheme.iniciais(nome), style: const TextStyle(color: Colors.white, fontWeight: FontWeight.w700)),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(nome, style: tema.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700), overflow: TextOverflow.ellipsis),
                      const SizedBox(height: 2),
                      Text(detalhes, style: tema.textTheme.bodyMedium?.copyWith(color: tema.colorScheme.onSurfaceVariant)),
                    ],
                  ),
                ),
                if (silenciado) Icon(Icons.volume_off, color: tema.colorScheme.onSurfaceVariant),
                const SizedBox(width: 4),
                Icon(Icons.chevron_right, color: tema.colorScheme.onSurfaceVariant),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _Vazio extends StatelessWidget {
  const _Vazio({required this.temEmpresa});

  final bool temEmpresa;

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 32, horizontal: 8),
      child: Column(
        children: [
          Icon(Icons.forum_outlined, size: 48, color: tema.colorScheme.onSurfaceVariant),
          const SizedBox(height: 12),
          Text(
            temEmpresa ? 'Nenhum canal disponível agora.' : 'Você ainda não está em nenhum canal.',
            style: tema.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 6),
          Text(
            temEmpresa
                ? 'Os canais liberados pela sua empresa aparecem aqui.'
                : 'Em breve você poderá criar canais e encontrar os da comunidade na aba Explorar.',
            style: tema.textTheme.bodyMedium?.copyWith(color: tema.colorScheme.onSurfaceVariant),
            textAlign: TextAlign.center,
          ),
        ],
      ),
    );
  }
}
