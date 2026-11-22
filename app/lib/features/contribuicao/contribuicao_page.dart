import 'package:flutter/material.dart';

import '../../core/http_client.dart';
import '../../core/theme.dart';
import '../../orbit/ponto_transmissao.dart';
import '../ponto/cartao_ponto_transmissao.dart';

/// Aba Contribuição (US7, FR-040): quantas leituras o celular fez — sem
/// mostrar quais Orbits foram lidos (FR-011) — e os controles do ponto de
/// transmissão e do bip.
class ContribuicaoPage extends StatefulWidget {
  const ContribuicaoPage({super.key});

  @override
  State<ContribuicaoPage> createState() => _ContribuicaoPageState();
}

class _ContribuicaoPageState extends State<ContribuicaoPage> {
  final _api = ApiClient();
  final _ponto = PontoTransmissao.instance;
  int? _hoje, _semana, _total;
  bool _erro = false;

  @override
  void initState() {
    super.initState();
    _carregar();
    // Cada leitura nova da sessão atualiza os números.
    _ponto.leituras.leiturasSessao.addListener(_carregarDepois);
  }

  @override
  void dispose() {
    _ponto.leituras.leiturasSessao.removeListener(_carregarDepois);
    super.dispose();
  }

  // O lote sai em até 15 s; espera um pouco para o servidor já ter contado.
  void _carregarDepois() => Future<void>.delayed(const Duration(seconds: 16), _carregar);

  Future<void> _carregar() async {
    try {
      final r = await _api.dio.get('/me/contribuicao');
      if (!mounted) return;
      setState(() {
        _hoje = r.data['hoje'] as int;
        _semana = r.data['semana'] as int;
        _total = r.data['total'] as int;
        _erro = false;
      });
    } catch (_) {
      if (mounted) setState(() => _erro = true);
    }
  }

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    return SafeArea(
      bottom: false,
      child: RefreshIndicator(
        onRefresh: _carregar,
        child: ListenableBuilder(
          listenable: _ponto,
          builder: (context, _) => ListView(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(4, 8, 4, 16),
                child: Text('Sua contribuição', style: tema.textTheme.headlineMedium?.copyWith(fontWeight: FontWeight.w700)),
              ),
              _CartaoEstado(ponto: _ponto),
              const SizedBox(height: 16),
              Row(
                children: [
                  Expanded(child: _Numero(valor: _hoje, rotulo: 'hoje', destaque: true)),
                  const SizedBox(width: 10),
                  Expanded(child: _Numero(valor: _semana, rotulo: 'na semana')),
                  const SizedBox(width: 10),
                  Expanded(child: _Numero(valor: _total, rotulo: 'no total')),
                ],
              ),
              if (_erro)
                Padding(
                  padding: const EdgeInsets.only(top: 8),
                  child: Text('Sem conexão: os números podem estar desatualizados.', style: TextStyle(color: tema.colorScheme.error)),
                ),
              const SizedBox(height: 16),
              Text(
                'Cada número é um caminhão que a KTM conseguiu localizar porque você passou perto. '
                'Você não vê quais foram — e a KTM não sabe que foi você.',
                style: tema.textTheme.bodyLarge?.copyWith(height: 1.5),
              ),
              const SizedBox(height: 16),
              Card(
                child: Column(
                  children: [
                    SwitchListTile(
                      secondary: const Icon(Icons.volume_up_outlined),
                      title: const Text('Som do bip'),
                      value: _ponto.bipSom,
                      onChanged: (v) => _ponto.definirBip(som: v),
                    ),
                    const Divider(),
                    SwitchListTile(
                      secondary: const Icon(Icons.vibration),
                      title: const Text('Vibração do bip'),
                      value: _ponto.bipVibracao,
                      onChanged: (v) => _ponto.definirBip(vibracao: v),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _CartaoEstado extends StatelessWidget {
  const _CartaoEstado({required this.ponto});

  final PontoTransmissao ponto;

  @override
  Widget build(BuildContext context) {
    final (titulo, texto) = switch (ponto.estado) {
      EstadoPonto.ativo => ('Ponto de transmissão ativo', 'Procurando caminhões com Orbit por perto.'),
      EstadoPonto.iniciando => ('Ligando…', 'Preparando Bluetooth e localização.'),
      EstadoPonto.suspenso => ('Ponto de transmissão pausado', ponto.mensagem ?? 'Toque para tentar de novo.'),
      EstadoPonto.desativado => ('Ponto de transmissão desligado', 'Ligue para ajudar a rede Orbit.'),
    };
    return Container(
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: AppTheme.fundoOrbit(context),
        border: Border.all(color: AppTheme.bordaOrbit(context)),
        borderRadius: BorderRadius.circular(18),
      ),
      child: Row(
        children: [
          Icon(ponto.estado == EstadoPonto.suspenso ? Icons.sensors_off : Icons.sensors, color: AppTheme.ambar, size: 30),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(titulo, style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700)),
                const SizedBox(height: 2),
                Text(texto, style: TextStyle(color: AppTheme.textoOrbit(context))),
                if (ponto.estado == EstadoPonto.suspenso)
                  TextButton(
                    style: TextButton.styleFrom(padding: EdgeInsets.zero, alignment: Alignment.centerLeft),
                    onPressed: ponto.tentarDeNovo,
                    child: const Text('Tentar de novo'),
                  ),
              ],
            ),
          ),
          Switch(
            value: ponto.ligado,
            activeTrackColor: const Color(0xFFC98512),
            onChanged: ponto.estado == EstadoPonto.iniciando
                ? null
                : (ligar) => ligar ? confirmarPontoTransmissao(context) : ponto.desativar(),
          ),
        ],
      ),
    );
  }
}

class _Numero extends StatelessWidget {
  const _Numero({required this.valor, required this.rotulo, this.destaque = false});

  final int? valor;
  final String rotulo;
  final bool destaque;

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              valor?.toString() ?? '—',
              style: tema.textTheme.headlineLarge?.copyWith(
                fontWeight: FontWeight.w700,
                color: destaque ? AppTheme.ambar : null,
              ),
            ),
            Text(rotulo, style: tema.textTheme.bodyMedium?.copyWith(color: tema.colorScheme.onSurfaceVariant)),
          ],
        ),
      ),
    );
  }
}
