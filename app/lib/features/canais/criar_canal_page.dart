import 'package:flutter/material.dart';
import 'package:geolocator/geolocator.dart';
import 'package:go_router/go_router.dart';

import '../../core/router.dart';
import '../../services/canais_api.dart';

/// Criar canal da comunidade (FR-029): nome, quem pode entrar, limite de
/// pessoas e, opcionalmente, uma área a partir de onde a pessoa está.
class CriarCanalPage extends StatefulWidget {
  const CriarCanalPage({super.key, this.api, this.obterPosicao});

  final CanaisApi? api;

  /// Permite aos testes trocar o GPS.
  final Future<Position> Function()? obterPosicao;

  @override
  State<CriarCanalPage> createState() => _CriarCanalPageState();
}

class _CriarCanalPageState extends State<CriarCanalPage> {
  final _form = GlobalKey<FormState>();
  final _nome = TextEditingController();
  final _descricao = TextEditingController();
  late final CanaisApi _api = widget.api ?? CanaisApi();

  bool _publico = false;
  int _limite = 10;
  bool _area = false;
  double _raioKm = 15;
  Position? _centro;
  bool _buscandoPosicao = false;
  bool _criando = false;
  String? _erro;

  @override
  void dispose() {
    _nome.dispose();
    _descricao.dispose();
    super.dispose();
  }

  Future<void> _alternarArea(bool ligar) async {
    if (!ligar) {
      setState(() => _area = false);
      return;
    }
    setState(() {
      _buscandoPosicao = true;
      _erro = null;
    });
    try {
      final obter = widget.obterPosicao ??
          () async {
            var permissao = await Geolocator.checkPermission();
            if (permissao == LocationPermission.denied) permissao = await Geolocator.requestPermission();
            if (permissao == LocationPermission.denied || permissao == LocationPermission.deniedForever) {
              throw StateError('sem permissão');
            }
            return Geolocator.getCurrentPosition(
              locationSettings: const LocationSettings(accuracy: LocationAccuracy.high, timeLimit: Duration(seconds: 10)),
            );
          };
      final posicao = await obter();
      if (mounted) {
        setState(() {
          _area = true;
          _centro = posicao;
        });
      }
    } catch (_) {
      if (mounted) setState(() => _erro = 'Não foi possível pegar sua localização. Ligue o GPS e permita o acesso.');
    } finally {
      if (mounted) setState(() => _buscandoPosicao = false);
    }
  }

  Future<void> _criar() async {
    if (_criando || !_form.currentState!.validate()) return;
    setState(() {
      _criando = true;
      _erro = null;
    });
    try {
      final canal = await _api.criar(
        nome: _nome.text.trim(),
        descricao: _descricao.text.trim(),
        publico: _publico,
        limite: _limite,
        latitude: _area ? _centro?.latitude : null,
        longitude: _area ? _centro?.longitude : null,
        raioMetros: _area ? (_raioKm * 1000).round() : null,
      );
      if (!mounted) return;
      if (canal.privado) {
        context.pushReplacement('/canal/${canal.id}/convite');
      } else {
        context.pushReplacement('/canal/${canal.id}', extra: AberturaCanal(nome: canal.nome));
      }
    } on ErroCanal catch (e) {
      if (mounted) setState(() => _erro = e.mensagem);
    } finally {
      if (mounted) setState(() => _criando = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('Novo canal')),
      body: Form(
        key: _form,
        child: ListView(
          padding: const EdgeInsets.all(20),
          children: [
            TextFormField(
              key: const Key('campo-nome-canal'),
              controller: _nome,
              textCapitalization: TextCapitalization.sentences,
              maxLength: 60,
              decoration: const InputDecoration(labelText: 'Nome do canal'),
              validator: (v) => (v ?? '').trim().isEmpty ? 'Dê um nome ao canal' : null,
            ),
            const SizedBox(height: 8),
            TextFormField(
              controller: _descricao,
              textCapitalization: TextCapitalization.sentences,
              maxLength: 140,
              decoration: const InputDecoration(labelText: 'Descrição (opcional)', hintText: 'Ex.: saída de Curitiba às 4h'),
            ),
            const SizedBox(height: 8),
            Text('Quem pode entrar', style: tema.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w700)),
            const SizedBox(height: 8),
            Row(
              children: [
                Expanded(
                  child: _Opcao(
                    titulo: 'Público',
                    texto: 'Aparece na busca',
                    selecionada: _publico,
                    aoTocar: () => setState(() => _publico = true),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: _Opcao(
                    titulo: 'Privado',
                    texto: 'Só com convite',
                    selecionada: !_publico,
                    aoTocar: () => setState(() => _publico = false),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 20),
            Text('Máximo de pessoas', style: tema.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w700)),
            const SizedBox(height: 8),
            SegmentedButton<int>(
              segments: [for (final n in const [5, 10, 15, 20]) ButtonSegment(value: n, label: Text('$n'))],
              selected: {_limite},
              showSelectedIcon: false,
              onSelectionChanged: (s) => setState(() => _limite = s.first),
            ),
            const SizedBox(height: 20),
            Card(
              child: Column(
                children: [
                  SwitchListTile(
                    secondary: const Icon(Icons.location_on_outlined),
                    title: const Text('Limitar a uma área'),
                    subtitle: Text(
                      _buscandoPosicao
                          ? 'Pegando sua localização…'
                          : _area
                          ? 'Raio de ${_raioKm.round()} km a partir de onde você está'
                          : 'Qualquer lugar do Brasil',
                    ),
                    value: _area,
                    onChanged: _buscandoPosicao ? null : _alternarArea,
                  ),
                  if (_area)
                    Padding(
                      padding: const EdgeInsets.fromLTRB(8, 0, 8, 8),
                      child: Slider(
                        value: _raioKm,
                        min: 1,
                        max: 50,
                        divisions: 49,
                        label: '${_raioKm.round()} km',
                        onChanged: (v) => setState(() => _raioKm = v),
                      ),
                    ),
                ],
              ),
            ),
            if (_erro != null) ...[
              const SizedBox(height: 12),
              Text(_erro!, style: TextStyle(color: tema.colorScheme.error)),
            ],
            const SizedBox(height: 24),
            FilledButton(
              key: const Key('botao-criar-canal'),
              style: FilledButton.styleFrom(minimumSize: const Size.fromHeight(56)),
              onPressed: _criando ? null : _criar,
              child: const Text('Criar canal'),
            ),
          ],
        ),
      ),
    );
  }
}

class _Opcao extends StatelessWidget {
  const _Opcao({required this.titulo, required this.texto, required this.selecionada, required this.aoTocar});

  final String titulo;
  final String texto;
  final bool selecionada;
  final VoidCallback aoTocar;

  @override
  Widget build(BuildContext context) {
    final cores = Theme.of(context).colorScheme;
    return Semantics(
      selected: selecionada,
      button: true,
      child: Material(
        color: selecionada ? cores.primary.withValues(alpha: 0.14) : cores.surfaceContainerHighest,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(14),
          side: BorderSide(color: selecionada ? cores.primary : cores.outlineVariant, width: 2),
        ),
        child: InkWell(
          borderRadius: BorderRadius.circular(14),
          onTap: aoTocar,
          child: Padding(
            padding: const EdgeInsets.all(14),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(titulo, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w700)),
                const SizedBox(height: 2),
                Text(texto, style: TextStyle(color: cores.onSurfaceVariant)),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
