import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/theme.dart';
import '../../orbit/ponto_transmissao.dart';

/// Casca com as 4 áreas principais do app (FR-038): qualquer uma fica a um
/// toque de distância. Azul nas abas da rádio, âmbar na de contribuição.
class CascaPage extends StatefulWidget {
  const CascaPage({super.key, required this.navegacao});

  final StatefulNavigationShell navegacao;

  @override
  State<CascaPage> createState() => _CascaPageState();
}

class _CascaPageState extends State<CascaPage> {
  @override
  void initState() {
    super.initState();
    // Religa o ponto de transmissão se o usuário o deixou ligado (US1).
    PontoTransmissao.instance.retomarSeAtivo();
  }

  @override
  Widget build(BuildContext context) {
    final navegacao = widget.navegacao;
    final naContribuicao = navegacao.currentIndex == 2;
    return Scaffold(
      body: navegacao,
      bottomNavigationBar: NavigationBarTheme(
        data: NavigationBarTheme.of(context).copyWith(
          indicatorColor: naContribuicao
              ? (Theme.of(context).brightness == Brightness.dark ? const Color(0xFF3A2A0B) : const Color(0xFFFCE6B8))
              : null,
        ),
        child: NavigationBar(
          selectedIndex: navegacao.currentIndex,
          onDestinationSelected: (i) => navegacao.goBranch(i, initialLocation: i == navegacao.currentIndex),
          destinations: const [
            NavigationDestination(icon: Icon(Icons.mic_none), selectedIcon: Icon(Icons.mic), label: 'Canais'),
            NavigationDestination(icon: Icon(Icons.explore_outlined), selectedIcon: Icon(Icons.explore), label: 'Explorar'),
            NavigationDestination(
              icon: Icon(Icons.sensors),
              selectedIcon: Icon(Icons.sensors, color: AppTheme.ambar),
              label: 'Contribuição',
            ),
            NavigationDestination(icon: Icon(Icons.person_outline), selectedIcon: Icon(Icons.person), label: 'Perfil'),
          ],
        ),
      ),
    );
  }
}
