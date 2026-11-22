import 'dart:io' show Platform;

import 'package:flutter/foundation.dart' show kIsWeb;
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:permission_handler/permission_handler.dart';

import '../../core/sessao_app.dart';
import '../../core/theme.dart';

class _Permissao {
  const _Permissao(this.nome, this.porque, this.icone, this.cor, this.permissoes);

  final String nome;
  final String porque;
  final IconData icone;
  final Color cor;
  final List<Permission> permissoes;
}

/// Permissões explicadas antes de pedir (FR-036). Cada uma tem seu botão;
/// negar uma só deixa de fora a função dela (FR-037). Pedir o Bluetooth
/// aqui não liga o ponto de transmissão — isso tem consentimento próprio.
class PermissoesPage extends StatefulWidget {
  const PermissoesPage({super.key});

  @override
  State<PermissoesPage> createState() => _PermissoesPageState();
}

class _PermissoesPageState extends State<PermissoesPage> with WidgetsBindingObserver {
  static final _itens = [
    const _Permissao('Microfone', 'Para você falar no canal ao segurar o botão.', Icons.mic_none, AppTheme.azulPrimario, [Permission.microphone]),
    const _Permissao(
      'Localização',
      'Para canais da sua região e para marcar onde um Orbit foi visto.',
      Icons.location_on_outlined,
      AppTheme.azulPrimario,
      [Permission.locationWhenInUse],
    ),
    _Permissao(
      'Bluetooth',
      'Para perceber caminhões com Orbit por perto, se você ativar o ponto de transmissão.',
      Icons.bluetooth,
      AppTheme.ambar,
      !kIsWeb && Platform.isAndroid ? [Permission.bluetoothScan, Permission.bluetoothConnect] : [Permission.bluetooth],
    ),
    const _Permissao(
      'Notificações',
      'Para avisar quando a rádio segue tocando em segundo plano.',
      Icons.notifications_none,
      AppTheme.azulPrimario,
      [Permission.notification],
    ),
  ];

  final _concedidas = <String, bool>{};

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _atualizar();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  // Ao voltar das configurações do sistema, relê o estado.
  @override
  void didChangeAppLifecycleState(AppLifecycleState estado) {
    if (estado == AppLifecycleState.resumed) _atualizar();
  }

  Future<void> _atualizar() async {
    for (final item in _itens) {
      var todas = true;
      for (final p in item.permissoes) {
        final s = await p.status;
        todas &= s.isGranted || s.isLimited;
      }
      _concedidas[item.nome] = todas;
    }
    if (mounted) setState(() {});
  }

  Future<void> _pedir(_Permissao item) async {
    final resultado = await item.permissoes.request();
    if (resultado.values.any((s) => s.isPermanentlyDenied)) {
      await openAppSettings();
    }
    await _atualizar();
  }

  Future<void> _continuar() async {
    await SessaoApp.instance.marcarPermissoesVistas();
    if (mounted) context.go('/canais');
  }

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    return Scaffold(
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(20, 32, 20, 24),
          children: [
            Text('Antes de começar', style: tema.textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.w700)),
            const SizedBox(height: 8),
            Text(
              'O app precisa de algumas permissões. Você pode negar qualquer uma: só a função dela fica de fora.',
              style: tema.textTheme.bodyLarge?.copyWith(color: tema.colorScheme.onSurfaceVariant),
            ),
            const SizedBox(height: 20),
            for (final item in _itens) ...[
              Card(
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(16, 14, 12, 14),
                  child: Row(
                    children: [
                      Container(
                        width: 44,
                        height: 44,
                        decoration: BoxDecoration(color: item.cor.withValues(alpha: 0.14), borderRadius: BorderRadius.circular(12)),
                        child: Icon(item.icone, color: item.cor),
                      ),
                      const SizedBox(width: 14),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(item.nome, style: tema.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700)),
                            const SizedBox(height: 2),
                            Text(item.porque, style: tema.textTheme.bodyMedium?.copyWith(color: tema.colorScheme.onSurfaceVariant)),
                          ],
                        ),
                      ),
                      const SizedBox(width: 8),
                      if (_concedidas[item.nome] ?? false)
                        const Row(
                          children: [
                            Icon(Icons.check, color: AppTheme.verdeFalando),
                            SizedBox(width: 4),
                            Text('Ok', style: TextStyle(color: AppTheme.verdeFalando, fontWeight: FontWeight.w700)),
                          ],
                        )
                      else
                        FilledButton(
                          style: FilledButton.styleFrom(minimumSize: const Size(88, 44)),
                          onPressed: () => _pedir(item),
                          child: const Text('Permitir'),
                        ),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 12),
            ],
            const SizedBox(height: 12),
            FilledButton(
              style: FilledButton.styleFrom(minimumSize: const Size.fromHeight(56)),
              onPressed: _continuar,
              child: const Text('Continuar'),
            ),
            const SizedBox(height: 8),
            Text(
              'Dá para mudar depois nas configurações do celular.',
              textAlign: TextAlign.center,
              style: tema.textTheme.bodySmall?.copyWith(color: tema.colorScheme.onSurfaceVariant),
            ),
          ],
        ),
      ),
    );
  }
}
