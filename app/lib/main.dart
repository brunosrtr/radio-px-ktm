import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:hive_flutter/hive_flutter.dart';

import 'core/rede_local.dart';
import 'core/router.dart';
import 'core/sessao_app.dart';
import 'core/theme.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await Hive.initFlutter();
  await SessaoApp.instance.carregar();
  await RedeLocal.instance.iniciar();
  runApp(const ProviderScope(child: RadioPxApp()));
}

class RadioPxApp extends StatefulWidget {
  const RadioPxApp({super.key, this.router});

  /// Permite aos testes começar em outra rota.
  final GoRouter? router;

  @override
  State<RadioPxApp> createState() => _RadioPxAppState();
}

class _RadioPxAppState extends State<RadioPxApp> {
  late final GoRouter _router = widget.router ?? criarRouter();

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'Rádio PX KTM',
      debugShowCheckedModeBanner: false,
      theme: AppTheme.light,
      darkTheme: AppTheme.dark,
      routerConfig: _router,
    );
  }
}
