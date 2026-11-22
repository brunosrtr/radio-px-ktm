import 'dart:async';

import 'package:flutter/material.dart';
import 'package:geolocator/geolocator.dart';
import 'package:go_router/go_router.dart';

import '../../core/router.dart';
import '../../services/canais_api.dart';
import '../canais/entrar_convite.dart';

/// Aba Explorar (FR-030): canais públicos da comunidade, pela busca por nome
/// ou pelos que têm área perto de você; e a entrada em privados por código
/// ou QR.
class ExplorarPage extends StatefulWidget {
  const ExplorarPage({super.key, this.api, this.obterPosicao});

  final CanaisApi? api;

  /// Permite aos testes trocar o GPS.
  final Future<Position?> Function()? obterPosicao;

  @override
  State<ExplorarPage> createState() => _ExplorarPageState();
}

class _ExplorarPageState extends State<ExplorarPage> {
  late final CanaisApi _api = widget.api ?? CanaisApi();
  final _busca = TextEditingController();
  Timer? _espera;
  bool _perto = true;
  bool _carregando = true;
  String? _erro;
  String? _aviso;
  List<Canal> _canais = [];

  @override
  void initState() {
    super.initState();
    _carregar();
  }

  @override
  void dispose() {
    _espera?.cancel();
    _busca.dispose();
    super.dispose();
  }

  Future<Position?> _posicao() async {
    if (widget.obterPosicao != null) return widget.obterPosicao!();
    try {
      final permissao = await Geolocator.checkPermission();
      if (permissao == LocationPermission.denied || permissao == LocationPermission.deniedForever) return null;
      return await Geolocator.getLastKnownPosition() ??
          await Geolocator.getCurrentPosition(
            locationSettings: const LocationSettings(accuracy: LocationAccuracy.medium, timeLimit: Duration(seconds: 8)),
          );
    } catch (_) {
      return null;
    }
  }

  Future<void> _carregar() async {
    setState(() {
      _carregando = true;
      _erro = null;
      _aviso = null;
    });
    final posicao = await _posicao();
    if (_perto && posicao == null) {
      _aviso = 'Sem localização: mostrando todos os canais públicos.';
    }
    try {
      final canais = await _api.explorar(
        busca: _busca.text.trim(),
        lat: posicao?.latitude,
        lng: posicao?.longitude,
        perto: _perto && posicao != null,
      );
      if (mounted) setState(() => _canais = canais);
    } on ErroCanal catch (e) {
      if (mounted) setState(() => _erro = e.mensagem);
    } finally {
      if (mounted) setState(() => _carregando = false);
    }
  }

  void _aoDigitar(String _) {
    _espera?.cancel();
    _espera = Timer(const Duration(milliseconds: 400), _carregar);
  }

  Future<void> _favoritar(Canal canal) async {
    try {
      await _api.favoritar(canal.id, !canal.favorito);
      await _carregar();
      if (mounted && !canal.favorito) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('"${canal.nome}" está nos seus canais.')));
      }
    } on ErroCanal catch (e) {
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(e.mensagem)));
    }
  }

  String _detalhe(Canal c) {
    final partes = ['${c.participantes} de ${c.limite}'];
    final d = c.distanciaMetros;
    if (d != null) partes.add(d < 1 ? 'você está na área' : 'a ${(d / 1000).toStringAsFixed(d < 10000 ? 1 : 0)} km');
    if (c.geocerca && c.raioMetros != null) partes.add('área de ${(c.raioMetros! / 1000).round()} km');
    return partes.join(' · ');
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
              child: Text('Explorar', style: tema.textTheme.headlineMedium?.copyWith(fontWeight: FontWeight.w700)),
            ),
            TextField(
              key: const Key('busca-canais'),
              controller: _busca,
              onChanged: _aoDigitar,
              textInputAction: TextInputAction.search,
              onSubmitted: (_) => _carregar(),
              decoration: const InputDecoration(prefixIcon: Icon(Icons.search), hintText: 'Buscar canal pelo nome'),
            ),
            const SizedBox(height: 12),
            Wrap(
              spacing: 8,
              children: [
                ChoiceChip(
                  label: const Text('Perto de mim'),
                  selected: _perto,
                  onSelected: (_) {
                    setState(() => _perto = true);
                    _carregar();
                  },
                ),
                ChoiceChip(
                  label: const Text('Todos os públicos'),
                  selected: !_perto,
                  onSelected: (_) {
                    setState(() => _perto = false);
                    _carregar();
                  },
                ),
              ],
            ),
            if (_aviso != null) ...[
              const SizedBox(height: 8),
              Text(_aviso!, style: tema.textTheme.bodyMedium?.copyWith(color: tema.colorScheme.onSurfaceVariant)),
            ],
            const SizedBox(height: 16),
            if (_carregando)
              const Padding(padding: EdgeInsets.all(32), child: Center(child: CircularProgressIndicator()))
            else if (_erro != null)
              Padding(padding: const EdgeInsets.all(16), child: Text(_erro!, textAlign: TextAlign.center))
            else if (_canais.isEmpty)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 24),
                child: Text(
                  _busca.text.isNotEmpty
                      ? 'Nenhum canal público com esse nome.'
                      : _perto
                      ? 'Nenhum canal com área perto de você. Veja todos os públicos ou crie um na aba Canais.'
                      : 'Ainda não há canais públicos. Crie o primeiro na aba Canais.',
                  textAlign: TextAlign.center,
                  style: tema.textTheme.bodyLarge?.copyWith(color: tema.colorScheme.onSurfaceVariant),
                ),
              )
            else
              for (final canal in _canais) ...[
                Card(
                  child: Padding(
                    padding: const EdgeInsets.fromLTRB(16, 12, 8, 12),
                    child: Row(
                      children: [
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(canal.nome, style: tema.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700)),
                              if (canal.descricao.isNotEmpty)
                                Text(canal.descricao, maxLines: 1, overflow: TextOverflow.ellipsis),
                              Text(_detalhe(canal), style: tema.textTheme.bodyMedium?.copyWith(color: tema.colorScheme.onSurfaceVariant)),
                            ],
                          ),
                        ),
                        IconButton(
                          tooltip: canal.favorito ? 'Tirar dos favoritos' : 'Favoritar',
                          onPressed: () => _favoritar(canal),
                          icon: Icon(canal.favorito ? Icons.star : Icons.star_border, color: canal.favorito ? Colors.amber : null),
                        ),
                        FilledButton(
                          style: FilledButton.styleFrom(minimumSize: const Size(72, 44)),
                          onPressed: () async {
                            await context.push('/canal/${canal.id}', extra: AberturaCanal(nome: canal.nome));
                            if (mounted) await _carregar();
                          },
                          child: const Text('Entrar'),
                        ),
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 8),
              ],
            const SizedBox(height: 8),
            OutlinedButton.icon(
              onPressed: () => abrirEntradaPorConvite(context, api: _api),
              icon: const Icon(Icons.qr_code_2),
              label: const Text('Entrar com código ou QR de convite'),
            ),
          ],
        ),
      ),
    );
  }
}
