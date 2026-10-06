import 'dart:async';

import 'package:flutter/material.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import '../../core/rede_local.dart';

class ConexaoLocal extends StatefulWidget {
  const ConexaoLocal({super.key});
  @override
  State<ConexaoLocal> createState() => _ConexaoLocalState();
}

class _ConexaoLocalState extends State<ConexaoLocal> {
  final _rede = RedeLocal.instance;
  bool _ocupado = false;
  @override
  void initState() {
    super.initState();
    _rede.addListener(_atualizar);
    if (_rede.inicializada) unawaited(_rede.procurar());
  }

  void _atualizar() {
    if (mounted) setState(() {});
  }

  Future<void> _escanear() async {
    final texto = await Navigator.of(context)
        .push<String>(MaterialPageRoute(builder: (_) => const _LeitorQr()));
    if (texto == null || !mounted) return;
    setState(() => _ocupado = true);
    try {
      await _rede.lerQr(texto);
    } catch (erro) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(
              erro is FormatException
                  ? erro.message
                  : 'Não foi possível conectar ao computador.',
            ),
          ),
        );
      }
    } finally {
      if (mounted) setState(() => _ocupado = false);
    }
  }

  Future<void> _selecionar(ServidorLocal servidor) async {
    setState(() => _ocupado = true);
    try {
      await _rede.selecionar(servidor);
    } catch (_) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Computador indisponível. Tente procurar novamente.'),
          ),
        );
      }
    } finally {
      if (mounted) setState(() => _ocupado = false);
    }
  }

  @override
  void dispose() {
    _rede.removeListener(_atualizar);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (RedeLocal.temUrlFixa) return const SizedBox.shrink();
    final ocupado = _rede.procurando || _ocupado;
    return Column(
      children: [
        const Icon(Icons.computer),
        Text(
          ocupado
              ? 'Procurando computador…'
              : _rede.servidor?.nome ?? 'Conecte ao computador da empresa',
          textAlign: TextAlign.center,
        ),
        if (_rede.erro != null)
          Padding(
            padding: const EdgeInsets.only(top: 8),
            child: Text(
              _rede.erro!,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ),
        for (final servidor in _rede.encontrados)
          if (_rede.servidor?.id != servidor.id)
            TextButton(
              onPressed: ocupado ? null : () => _selecionar(servidor),
              child: Text(
                '${servidor.nome} (${servidor.id.substring(0, servidor.id.length < 8 ? servidor.id.length : 8)})',
              ),
            ),
        Wrap(
          alignment: WrapAlignment.center,
          children: [
            TextButton.icon(
              onPressed: ocupado ? null : _rede.procurar,
              icon: const Icon(Icons.refresh),
              label: const Text('Procurar'),
            ),
            TextButton.icon(
              onPressed: ocupado ? null : _escanear,
              icon: const Icon(Icons.qr_code_scanner),
              label: const Text('Ler QR Code'),
            ),
          ],
        ),
        const SizedBox(height: 12),
      ],
    );
  }
}

class _LeitorQr extends StatefulWidget {
  const _LeitorQr();
  @override
  State<_LeitorQr> createState() => _LeitorQrState();
}

class _LeitorQrState extends State<_LeitorQr> {
  final _controller = MobileScannerController(formats: [BarcodeFormat.qrCode]);
  bool _lido = false;

  @override
  void dispose() {
    unawaited(_controller.dispose());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Leia o QR Code do painel')),
    body: MobileScanner(
      controller: _controller,
      onDetect: (captura) {
        if (_lido) return;
        for (final codigo in captura.barcodes) {
          final valor = codigo.rawValue;
          if (valor == null) continue;
          _lido = true;
          Navigator.of(context).pop(valor);
          return;
        }
      },
      errorBuilder: (context, error) => const Center(
        child: Padding(
          padding: EdgeInsets.all(24),
          child: Text(
            'Não foi possível abrir a câmera. Permita o acesso à câmera nas configurações e tente novamente.',
          ),
        ),
      ),
    ),
  );
}
