import 'package:flutter/material.dart';

/// Tema visual único do app — motoristas usam em cabine, muitas vezes sob luz
/// solar direta, daí o contraste alto e os alvos de toque grandes herdados do
/// `ThemeData` padrão do Material 3.
class AppTheme {
  AppTheme._();

  static ThemeData get light => ThemeData(
    useMaterial3: true,
    colorScheme: ColorScheme.fromSeed(seedColor: Colors.deepOrange),
  );

  static ThemeData get dark => ThemeData(
    useMaterial3: true,
    colorScheme: ColorScheme.fromSeed(
      seedColor: Colors.deepOrange,
      brightness: Brightness.dark,
    ),
  );
}
