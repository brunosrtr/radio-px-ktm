import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'core/routes.dart';
import 'core/theme.dart';
import 'features/auth/login_page.dart';
import 'features/canais/lista_canais_page.dart';

void main() {
  runApp(const ProviderScope(child: RadioPxApp()));
}

class RadioPxApp extends StatelessWidget {
  const RadioPxApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Rádio PX Digital',
      theme: AppTheme.light,
      darkTheme: AppTheme.dark,
      initialRoute: AppRoutes.login,
      routes: {
        AppRoutes.login: (_) => const LoginPage(),
        AppRoutes.canais: (_) => const ListaCanaisPage(),
      },
    );
  }
}
