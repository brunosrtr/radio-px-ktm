import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import '../../core/router.dart';
import '../../services/canais_api.dart';

/// Abre a folha "Entrar com código ou QR" (FR-030). Ao dar certo, abre o
/// canal.
Future<void> abrirEntradaPorConvite(BuildContext context, {CanaisApi? api}) async {
  final canal = await showModalBottomSheet<Canal>(
    context: context,
    isScrollControlled: true,
    showDragHandle: true,
    builder: (_) => _FolhaConvite(api: api ?? CanaisApi()),
  );
  if (canal != null && context.mounted) {
    await context.push('/canal/${canal.id}', extra: AberturaCanal(nome: canal.nome));
  }
}

class _FolhaConvite extends StatefulWidget {
  const _FolhaConvite({required this.api});

  final CanaisApi api;

  @override
  State<_FolhaConvite> createState() => _FolhaConviteState();
}

class _FolhaConviteState extends State<_FolhaConvite> {
  final _codigo = TextEditingController();
  bool _enviando = false;
  String? _erro;

  @override
  void dispose() {
    _codigo.dispose();
    super.dispose();
  }

  Future<void> _entrar(String entrada) async {
    final codigo = normalizarCodigoConvite(entrada);
    if (codigo == null) {
      setState(() => _erro = 'Código inválido. Ele tem 8 letras e números, como K7QM-2XPA.');
      return;
    }
    setState(() {
      _enviando = true;
      _erro = null;
    });
    try {
      final canal = await widget.api.entrarPorConvite(codigo);
      if (mounted) Navigator.of(context).pop(canal);
    } on ErroCanal catch (e) {
      if (mounted) setState(() => _erro = e.mensagem);
    } finally {
      if (mounted) setState(() => _enviando = false);
    }
  }

  Future<void> _lerQr() async {
    final lido = await Navigator.of(context).push<String>(MaterialPageRoute(builder: (_) => const LeitorQrConvite()));
    if (lido != null) {
      _codigo.text = normalizarCodigoConvite(lido) ?? lido;
      await _entrar(lido);
    }
  }

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    return Padding(
      padding: EdgeInsets.fromLTRB(20, 0, 20, 20 + MediaQuery.of(context).viewInsets.bottom),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text('Entrar num canal privado', style: tema.textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w700)),
          const SizedBox(height: 6),
          Text('Use o código que o criador passou ou leia o QR no celular dele.', style: tema.textTheme.bodyMedium),
          const SizedBox(height: 16),
          TextField(
            key: const Key('campo-codigo-convite'),
            controller: _codigo,
            autofocus: true,
            textCapitalization: TextCapitalization.characters,
            inputFormatters: [LengthLimitingTextInputFormatter(9)],
            style: const TextStyle(fontSize: 22, letterSpacing: 3, fontWeight: FontWeight.w700),
            decoration: const InputDecoration(labelText: 'Código', hintText: 'XXXX-XXXX'),
            onSubmitted: _entrar,
          ),
          if (_erro != null) ...[
            const SizedBox(height: 8),
            Text(_erro!, style: TextStyle(color: tema.colorScheme.error)),
          ],
          const SizedBox(height: 16),
          FilledButton(onPressed: _enviando ? null : () => _entrar(_codigo.text), child: const Text('Entrar')),
          const SizedBox(height: 8),
          OutlinedButton.icon(
            onPressed: _enviando ? null : _lerQr,
            icon: const Icon(Icons.qr_code_scanner),
            label: const Text('Ler QR Code'),
          ),
        ],
      ),
    );
  }
}

/// Câmera que lê o QR do convite e devolve o texto lido.
class LeitorQrConvite extends StatefulWidget {
  const LeitorQrConvite({super.key});

  @override
  State<LeitorQrConvite> createState() => _LeitorQrConviteState();
}

class _LeitorQrConviteState extends State<LeitorQrConvite> {
  final _controller = MobileScannerController(formats: [BarcodeFormat.qrCode]);
  bool _lido = false;

  @override
  void dispose() {
    unawaited(_controller.dispose());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Leia o QR do convite')),
    body: MobileScanner(
      controller: _controller,
      onDetect: (captura) {
        if (_lido) return;
        for (final codigo in captura.barcodes) {
          final valor = codigo.rawValue;
          if (valor == null || normalizarCodigoConvite(valor) == null) continue;
          _lido = true;
          Navigator.of(context).pop(valor);
          return;
        }
      },
      errorBuilder: (context, error) => const Center(
        child: Padding(
          padding: EdgeInsets.all(24),
          child: Text('Não foi possível abrir a câmera. Permita o acesso à câmera nas configurações e tente de novo.'),
        ),
      ),
    ),
  );
}

/// Tela aberta pelo link `radiopx://app/convite/<código>` (QR lido pela
/// câmera do celular): entra no canal e já abre a conversa.
class EntrarPorLinkPage extends StatefulWidget {
  const EntrarPorLinkPage({super.key, required this.codigo, this.api});

  final String codigo;
  final CanaisApi? api;

  @override
  State<EntrarPorLinkPage> createState() => _EntrarPorLinkPageState();
}

class _EntrarPorLinkPageState extends State<EntrarPorLinkPage> {
  String? _erro;

  @override
  void initState() {
    super.initState();
    _entrar();
  }

  Future<void> _entrar() async {
    final codigo = normalizarCodigoConvite(widget.codigo);
    if (codigo == null) {
      setState(() => _erro = 'Este link de convite não é válido.');
      return;
    }
    try {
      final canal = await (widget.api ?? CanaisApi()).entrarPorConvite(codigo);
      if (!mounted) return;
      context.go('/canais');
      unawaited(context.push('/canal/${canal.id}', extra: AberturaCanal(nome: canal.nome)));
    } on ErroCanal catch (e) {
      if (mounted) setState(() => _erro = e.mensagem);
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Convite')),
    body: Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: _erro == null
            ? const Column(mainAxisSize: MainAxisSize.min, children: [CircularProgressIndicator(), SizedBox(height: 16), Text('Entrando no canal…')])
            : Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(Icons.link_off, size: 44, color: Theme.of(context).colorScheme.error),
                  const SizedBox(height: 12),
                  Text(_erro!, textAlign: TextAlign.center, style: Theme.of(context).textTheme.titleMedium),
                  const SizedBox(height: 20),
                  FilledButton(onPressed: () => context.go('/canais'), child: const Text('Ir para os canais')),
                ],
              ),
      ),
    ),
  );
}
