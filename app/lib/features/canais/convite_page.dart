import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:share_plus/share_plus.dart';

import '../../services/canais_api.dart';

/// Convite de um canal privado (US5): QR Code, código, copiar,
/// compartilhar e gerar um código novo (FR-031). Só para quem gerencia.
class ConvitePage extends StatefulWidget {
  const ConvitePage({super.key, required this.canalId, this.api});

  final String canalId;
  final CanaisApi? api;

  @override
  State<ConvitePage> createState() => _ConvitePageState();
}

class _ConvitePageState extends State<ConvitePage> {
  late final CanaisApi _api = widget.api ?? CanaisApi();
  Canal? _canal;
  String? _codigo;
  String? _erro;
  bool _ocupado = false;

  @override
  void initState() {
    super.initState();
    _carregar();
  }

  Future<void> _carregar() async {
    try {
      final canal = await _api.detalhar(widget.canalId);
      if (!mounted) return;
      setState(() {
        _canal = canal;
        _codigo = canal.codigoConvite;
        if (_codigo == null) _erro = 'Este canal não usa convite.';
      });
    } on ErroCanal catch (e) {
      if (mounted) setState(() => _erro = e.mensagem);
    }
  }

  String _texto() =>
      'Entre no canal "${_canal?.nome}" na Rádio PX KTM com o código $_codigo '
      'ou pelo link ${linkConvite(_codigo!)}';

  Future<void> _gerarNovo() async {
    final confirmou = await showDialog<bool>(
      context: context,
      builder: (c) => AlertDialog(
        title: const Text('Gerar novo código?'),
        content: const Text('O código atual para de funcionar. Quem já está no canal continua.'),
        actions: [
          TextButton(onPressed: () => Navigator.of(c).pop(false), child: const Text('Cancelar')),
          FilledButton(onPressed: () => Navigator.of(c).pop(true), child: const Text('Gerar')),
        ],
      ),
    );
    if (confirmou != true) return;
    setState(() => _ocupado = true);
    try {
      final novo = await _api.novoConvite(widget.canalId);
      if (mounted) setState(() => _codigo = novo);
    } on ErroCanal catch (e) {
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(e.mensagem)));
    } finally {
      if (mounted) setState(() => _ocupado = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(_canal == null ? 'Convite' : 'Convidar para ${_canal!.nome}')),
      body: _erro != null
          ? Center(child: Padding(padding: const EdgeInsets.all(24), child: Text(_erro!, textAlign: TextAlign.center)))
          : _codigo == null
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(20),
              children: [
                Text(
                  'Canal privado: só entra quem tiver este código ou ler o QR Code no seu celular.',
                  style: tema.textTheme.bodyLarge,
                ),
                const SizedBox(height: 20),
                Center(
                  child: Container(
                    padding: const EdgeInsets.all(16),
                    decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(20)),
                    child: QrImageView(
                      data: linkConvite(_codigo!),
                      size: 228,
                      semanticsLabel: 'QR Code do convite',
                    ),
                  ),
                ),
                const SizedBox(height: 20),
                Text(
                  'CÓDIGO',
                  textAlign: TextAlign.center,
                  style: tema.textTheme.labelMedium?.copyWith(letterSpacing: 1.2, fontWeight: FontWeight.w700),
                ),
                SelectableText(
                  _codigo!,
                  textAlign: TextAlign.center,
                  style: tema.textTheme.displaySmall?.copyWith(fontWeight: FontWeight.w700, letterSpacing: 4),
                ),
                const SizedBox(height: 20),
                Row(
                  children: [
                    Expanded(
                      child: OutlinedButton.icon(
                        onPressed: () async {
                          await Clipboard.setData(ClipboardData(text: _codigo!));
                          if (context.mounted) {
                            ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Código copiado.')));
                          }
                        },
                        icon: const Icon(Icons.copy),
                        label: const Text('Copiar'),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: FilledButton.icon(
                        onPressed: () => SharePlus.instance.share(ShareParams(text: _texto())),
                        icon: const Icon(Icons.share),
                        label: const Text('Compartilhar'),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 24),
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text('Alguém saiu do grupo?', style: tema.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700)),
                        const SizedBox(height: 4),
                        const Text('Gere um código novo: o antigo para de funcionar. Quem já está no canal continua.'),
                        TextButton(onPressed: _ocupado ? null : _gerarNovo, child: const Text('Gerar novo código')),
                      ],
                    ),
                  ),
                ),
              ],
            ),
    );
  }
}
