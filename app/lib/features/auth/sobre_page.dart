import 'package:flutter/material.dart';

import '../../core/theme.dart';

class SobrePage extends StatelessWidget {
  const SobrePage({super.key});
  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Sobre a KTM')),
    body: SingleChildScrollView(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Container(
            color: const Color(0xFF080D12),
            padding: const EdgeInsets.all(28),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Image.asset(
                  'assets/branding/logo-ktm.png',
                  width: 150,
                  semanticLabel: 'KTM Indústria',
                ),
                const SizedBox(height: 32),
                const Text(
                  'TECNOLOGIA E SEGURANÇA VEICULAR',
                  style: TextStyle(
                    color: AppTheme.azulPrimario,
                    fontSize: 10,
                    letterSpacing: 1.5,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 16),
                const Text(
                  'Ao lado de quem move o Brasil.',
                  style: TextStyle(
                    color: Colors.white,
                    fontSize: 32,
                    height: 1.15,
                    fontWeight: FontWeight.w700,
                    letterSpacing: -1,
                  ),
                ),
              ],
            ),
          ),
          Padding(
            padding: const EdgeInsets.all(28),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Conheça a KTM Indústria',
                  style: Theme.of(context).textTheme.titleLarge
                      ?.copyWith(fontWeight: FontWeight.w700),
                ),
                const SizedBox(height: 16),
                const Text(
                  'Indústria brasileira de segurança veicular e gestão de frotas pesadas, com mais de três décadas de experiência. Desenvolve tecnologia própria para rastreamento, proteção embarcada, telemetria e integração de sistemas.',
                  style: TextStyle(height: 1.7),
                ),
                const SizedBox(height: 28),
                const ListTile(
                  contentPadding: EdgeInsets.zero,
                  leading: Icon(
                    Icons.route_outlined,
                    color: AppTheme.azulSecundario,
                  ),
                  title: Text('Tecnologia para o transporte'),
                  subtitle: Text('Proteção de veículos e cargas.'),
                ),
                const Divider(height: 28),
                const ListTile(
                  contentPadding: EdgeInsets.zero,
                  leading: Icon(
                    Icons.groups_outlined,
                    color: AppTheme.azulSecundario,
                  ),
                  title: Text('Atendimento próximo'),
                  subtitle: Text(
                    'Soluções para operações de diferentes portes.',
                  ),
                ),
                const SizedBox(height: 28),
                Text(
                  'Site institucional',
                  style: Theme.of(context).textTheme.titleSmall,
                ),
                const SizedBox(height: 8),
                const SelectableText(
                  'https://ktm.ind.br/',
                  style: TextStyle(
                    color: AppTheme.azulSecundario,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(height: 12),
                Text(
                  'Informações baseadas no site oficial da KTM Indústria.',
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ],
            ),
          ),
        ],
      ),
    ),
  );
}
