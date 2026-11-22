import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:permission_handler/permission_handler.dart';

import '../../core/theme.dart';
import '../../orbit/ponto_transmissao.dart';
import '../auth/documentos_legais.dart';

/// Cartão do ponto de transmissão (US1), no âmbar da rede Orbit: convida a
/// ativar, mostra o estado e as leituras da sessão — sem nenhum dado do
/// Orbit lido (FR-011). Tocar leva à aba Contribuição.
class CartaoPontoTransmissao extends StatelessWidget {
  const CartaoPontoTransmissao({super.key});

  @override
  Widget build(BuildContext context) {
    final ponto = PontoTransmissao.instance;
    return ListenableBuilder(
      listenable: Listenable.merge([ponto, ponto.leituras.leiturasSessao, ponto.orbitVirtual.anunciando]),
      builder: (context, _) {
        final orbitVirtual = ponto.orbitVirtual.anunciando.value;
        final leituras = ponto.leituras.leiturasSessao.value;
        return Column(
          children: [
            if (orbitVirtual != null) ...[const _AvisoOrbitVirtual(), const SizedBox(height: 10)],
            Material(
              color: AppTheme.fundoOrbit(context),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(16),
                side: BorderSide(color: AppTheme.bordaOrbit(context)),
              ),
              clipBehavior: Clip.antiAlias,
              child: switch (ponto.estado) {
                EstadoPonto.desativado => Padding(padding: const EdgeInsets.all(16), child: _Convite(mensagem: ponto.mensagem)),
                EstadoPonto.iniciando => const Padding(
                  padding: EdgeInsets.all(16),
                  child: Row(
                    children: [
                      SizedBox(width: 22, height: 22, child: CircularProgressIndicator(strokeWidth: 2, color: AppTheme.ambar)),
                      SizedBox(width: 14),
                      Text('Ligando o ponto de transmissão…'),
                    ],
                  ),
                ),
                EstadoPonto.ativo => InkWell(
                  onTap: () => context.go('/contribuicao'),
                  child: Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
                    child: Row(
                      children: [
                        const Icon(Icons.sensors, color: AppTheme.ambar, size: 28),
                        const SizedBox(width: 12),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              const Text('Ponto de transmissão ativo', style: TextStyle(fontWeight: FontWeight.w700, fontSize: 16)),
                              Text(
                                leituras == 0
                                    ? 'Procurando caminhões com Orbit por perto.'
                                    : 'Você ajudou a localizar $leituras ${leituras == 1 ? 'caminhão' : 'caminhões'} nesta sessão.',
                                style: TextStyle(color: AppTheme.textoOrbit(context)),
                              ),
                            ],
                          ),
                        ),
                        const Icon(Icons.chevron_right, color: AppTheme.ambar),
                      ],
                    ),
                  ),
                ),
                EstadoPonto.suspenso => Padding(
                  padding: const EdgeInsets.fromLTRB(14, 8, 8, 8),
                  child: Row(
                    children: [
                      const Icon(Icons.sensors_off, color: AppTheme.ambar, size: 28),
                      const SizedBox(width: 12),
                      Expanded(child: Text(ponto.mensagem ?? 'Ponto de transmissão pausado.')),
                      TextButton(
                        onPressed: () async {
                          if (ponto.mensagem?.startsWith('Permita') ?? false) {
                            if (await Permission.bluetoothScan.isPermanentlyDenied ||
                                await Permission.locationWhenInUse.isPermanentlyDenied) {
                              await openAppSettings();
                              return;
                            }
                          }
                          await ponto.tentarDeNovo();
                        },
                        child: const Text('Tentar de novo'),
                      ),
                    ],
                  ),
                ),
              },
            ),
          ],
        );
      },
    );
  }
}

class _Convite extends StatelessWidget {
  const _Convite({this.mensagem});

  final String? mensagem;

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            const Icon(Icons.sensors, color: AppTheme.ambar),
            const SizedBox(width: 8),
            Text('Ajude a rede Orbit', style: tema.textTheme.titleMedium),
          ],
        ),
        const SizedBox(height: 6),
        Text(
          'Seu celular pode avisar a KTM quando passar perto de um caminhão com Orbit. '
          'Funciona sozinho, em segundo plano.',
          style: tema.textTheme.bodyMedium,
        ),
        if (mensagem != null) ...[
          const SizedBox(height: 6),
          Text(mensagem!, style: TextStyle(color: tema.colorScheme.error)),
        ],
        const SizedBox(height: 10),
        Align(
          alignment: Alignment.centerRight,
          child: FilledButton.icon(
            style: FilledButton.styleFrom(backgroundColor: AppTheme.ambar, foregroundColor: const Color(0xFF1A1205)),
            onPressed: () => confirmarPontoTransmissao(context),
            icon: const Icon(Icons.sensors),
            label: const Text('Ativar'),
          ),
        ),
      ],
    );
  }
}

/// Explica a função antes de pedir o consentimento e as permissões
/// (FR-028, FR-036).
Future<void> confirmarPontoTransmissao(BuildContext context) async {
  final aceitou = await showDialog<bool>(
    context: context,
    builder: (contexto) => AlertDialog(
      icon: const Icon(Icons.sensors),
      title: const Text('Ser um ponto de transmissão'),
      content: SingleChildScrollView(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            const Text(
              'Quando você passar perto de um caminhão com o equipamento Orbit, seu celular avisa a KTM '
              'onde o caminhão foi visto. Você ouve um bip curto a cada aviso.',
            ),
            const SizedBox(height: 12),
            const _Item(Icons.visibility_off_outlined, 'Você não vê nenhum dado do caminhão.'),
            const _Item(Icons.person_off_outlined, 'A KTM não sabe quem fez a leitura.'),
            const _Item(Icons.bluetooth, 'Usamos Bluetooth e localização para isso, também com a tela apagada.'),
            const _Item(Icons.toggle_off_outlined, 'Você pode desligar quando quiser, sem perder a rádio.'),
            const SizedBox(height: 8),
            TextButton(
              style: TextButton.styleFrom(padding: EdgeInsets.zero),
              onPressed: () => abrirDocumentoLegal(contexto, '/privacidade'),
              child: const Text('Ler a política de privacidade'),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(onPressed: () => Navigator.of(contexto).pop(false), child: const Text('Agora não')),
        FilledButton(onPressed: () => Navigator.of(contexto).pop(true), child: const Text('Concordo, ativar')),
      ],
    ),
  );
  if (aceitou == true) await PontoTransmissao.instance.ativar();
}

class _Item extends StatelessWidget {
  const _Item(this.icone, this.texto);

  final IconData icone;
  final String texto;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 4),
    child: Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Icon(icone, size: 20, color: Theme.of(context).colorScheme.primary),
        const SizedBox(width: 10),
        Expanded(child: Text(texto)),
      ],
    ),
  );
}

/// Aviso permanente enquanto o aparelho se passa por um Orbit (FR-014).
class _AvisoOrbitVirtual extends StatelessWidget {
  const _AvisoOrbitVirtual();

  @override
  Widget build(BuildContext context) {
    final ponto = PontoTransmissao.instance;
    return MaterialBanner(
      backgroundColor: const Color(0xFFFFF3CD),
      leading: const Icon(Icons.local_shipping, color: Color(0xFF8A6100)),
      content: Text(
        'Este celular está se passando pelo ${ponto.orbitVirtual.anunciando.value} (demonstração).',
        style: const TextStyle(color: Color(0xFF5C4100)),
      ),
      actions: [TextButton(onPressed: ponto.pararOrbitVirtual, child: const Text('Parar'))],
    );
  }
}
