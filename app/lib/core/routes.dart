import 'package:flutter/material.dart';

/// Nomes de rota centralizados — as telas de `features/auth` e
/// `features/canais` (próximas fases) se registram aqui em vez de espalhar
/// strings literais pelo app.
class AppRoutes {
  AppRoutes._();

  static const String splash = '/';
  static const String login = '/login';
  static const String canais = '/canais';
  static const String canalAtivo = '/canais/ativo';
}

/// Placeholder exibido enquanto as telas de autenticação e canais (fases
/// seguintes) ainda não existem.
class SplashPage extends StatelessWidget {
  const SplashPage({super.key});

  @override
  Widget build(BuildContext context) {
    return const Scaffold(body: Center(child: Text('Rádio PX Digital')));
  }
}
