import 'package:flutter/material.dart';

/// Aba Explorar. A busca e os canais públicos próximos chegam com os canais
/// da comunidade (Etapa 6 da 003-rede-orbit); até lá a aba diz isso com
/// clareza, sem controles que não funcionam.
class ExplorarPage extends StatelessWidget {
  const ExplorarPage({super.key});

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    return SafeArea(
      bottom: false,
      child: ListView(
        padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(4, 8, 4, 24),
            child: Text('Explorar', style: tema.textTheme.headlineMedium?.copyWith(fontWeight: FontWeight.w700)),
          ),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                children: [
                  Icon(Icons.explore_outlined, size: 56, color: tema.colorScheme.primary),
                  const SizedBox(height: 16),
                  Text(
                    'Em breve: canais da comunidade',
                    textAlign: TextAlign.center,
                    style: tema.textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w700),
                  ),
                  const SizedBox(height: 8),
                  Text(
                    'Aqui você vai buscar canais pelo nome, ver os canais públicos perto de você e entrar em canais privados com um código ou QR de convite.',
                    textAlign: TextAlign.center,
                    style: tema.textTheme.bodyLarge?.copyWith(color: tema.colorScheme.onSurfaceVariant, height: 1.45),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}
