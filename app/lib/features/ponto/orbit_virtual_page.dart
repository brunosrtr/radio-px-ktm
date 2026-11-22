import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../orbit/orbit_sinal.dart';
import '../../orbit/orbit_virtual.dart';
import '../../orbit/ponto_transmissao.dart';

/// Modo Orbit virtual (US2): só para contas `admin_ktm` e `demo`. O celular
/// passa a anunciar o sinal de um Orbit para a demonstração à KTM.
class OrbitVirtualPage extends StatefulWidget {
  const OrbitVirtualPage({super.key});

  @override
  State<OrbitVirtualPage> createState() => _OrbitVirtualPageState();
}

class _OrbitVirtualPageState extends State<OrbitVirtualPage> {
  static const _demonstracao = ['ORBIT-0A1B2C', 'ORBIT-0A1B2D', 'ORBIT-0A1B2E'];
  final _id = TextEditingController(text: _demonstracao.first);
  bool _ocupado = false;
  String? _erro;

  @override
  void dispose() {
    _id.dispose();
    super.dispose();
  }

  Future<void> _alternar() async {
    final ponto = PontoTransmissao.instance;
    setState(() {
      _ocupado = true;
      _erro = null;
    });
    try {
      if (ponto.orbitVirtual.anunciando.value != null) {
        await ponto.pararOrbitVirtual();
      } else {
        final id = _id.text.trim().toUpperCase();
        if (!OrbitSinal.idValido(id)) {
          throw StateError('Use ORBIT- seguido de 6 letras ou números de 0–9 e A–F.');
        }
        await ponto.iniciarOrbitVirtual(id);
      }
    } on StateError catch (e) {
      _erro = e.message;
    } catch (e) {
      _erro = 'Não foi possível anunciar: $e';
    } finally {
      if (mounted) setState(() => _ocupado = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    final ponto = PontoTransmissao.instance;
    return Scaffold(
      appBar: AppBar(title: const Text('Orbit virtual')),
      body: ValueListenableBuilder<String?>(
        valueListenable: ponto.orbitVirtual.anunciando,
        builder: (context, anunciando, _) => ListView(
          padding: const EdgeInsets.all(20),
          children: [
            Container(
              padding: const EdgeInsets.all(20),
              decoration: BoxDecoration(
                color: anunciando != null ? const Color(0xFFFFF3CD) : tema.colorScheme.surfaceContainerHighest,
                borderRadius: BorderRadius.circular(16),
              ),
              child: Column(
                children: [
                  Icon(
                    anunciando != null ? Icons.local_shipping : Icons.local_shipping_outlined,
                    size: 56,
                    color: anunciando != null ? const Color(0xFF8A6100) : tema.colorScheme.onSurfaceVariant,
                  ),
                  const SizedBox(height: 10),
                  Text(
                    anunciando != null ? 'Anunciando $anunciando' : 'Parado',
                    style: tema.textTheme.titleLarge?.copyWith(
                      fontWeight: FontWeight.w700,
                      color: anunciando != null ? const Color(0xFF5C4100) : null,
                    ),
                  ),
                  const SizedBox(height: 6),
                  Text(
                    anunciando != null
                        ? 'Aproxime outro celular com o app: ele vai bipar e o caminhão aparece no painel Rede Orbit.'
                        : 'Este celular vai se passar por um Orbit instalado num caminhão.',
                    textAlign: TextAlign.center,
                    style: TextStyle(color: anunciando != null ? const Color(0xFF5C4100) : null),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 20),
            TextField(
              controller: _id,
              enabled: anunciando == null && !_ocupado,
              textCapitalization: TextCapitalization.characters,
              inputFormatters: [LengthLimitingTextInputFormatter(12)],
              decoration: const InputDecoration(labelText: 'Identificação do Orbit', prefixIcon: Icon(Icons.tag)),
            ),
            const SizedBox(height: 8),
            Wrap(
              spacing: 8,
              children: [
                for (final id in _demonstracao)
                  ChoiceChip(
                    label: Text(id),
                    selected: _id.text == id,
                    onSelected: anunciando == null ? (_) => setState(() => _id.text = id) : null,
                  ),
              ],
            ),
            if (!OrbitVirtual.plataformaSuportada) ...[
              const SizedBox(height: 16),
              Text(
                'O modo Orbit virtual só funciona em Android: o iPhone não permite anunciar este tipo de sinal.',
                style: TextStyle(color: tema.colorScheme.error),
              ),
            ],
            if (_erro != null) ...[
              const SizedBox(height: 16),
              Text(_erro!, style: TextStyle(color: tema.colorScheme.error)),
            ],
            const SizedBox(height: 20),
            FilledButton.icon(
              style: FilledButton.styleFrom(minimumSize: const Size.fromHeight(56)),
              onPressed: _ocupado || !OrbitVirtual.plataformaSuportada ? null : _alternar,
              icon: Icon(anunciando != null ? Icons.stop : Icons.play_arrow),
              label: Text(anunciando != null ? 'Parar de anunciar' : 'Começar a anunciar'),
            ),
            const SizedBox(height: 12),
            Text(
              'Para a demonstração, deixe este celular com a tela ligada ou com a notificação do app visível.',
              style: tema.textTheme.bodySmall,
              textAlign: TextAlign.center,
            ),
          ],
        ),
      ),
    );
  }
}
