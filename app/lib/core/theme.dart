import 'package:flutter/material.dart';

/// Tema visual único do app — motoristas usam em cabine, muitas vezes sob luz
/// solar direta, daí o contraste alto e os alvos de toque grandes herdados do
/// `ThemeData` padrão do Material 3. Paleta azul, no mesmo espírito visual do
/// painel web da empresa.
class AppTheme {
  AppTheme._();

  static const azulPrimario = Color(0xFF41C1F0);
  static const azulSecundario = Color(0xFF087FA9);

  static const paletaAvatares = [
    Color(0xFF149BC7),
    Color(0xFF31576A),
    Color(0xFF1C7C94),
    Color(0xFF48879C),
    Color(0xFF258B9B),
    Color(0xFF304C60),
  ];

  static Color corDoAvatar(String nome) {
    var hash = 0;
    for (final unidade in nome.codeUnits) {
      hash = unidade + ((hash << 5) - hash);
    }
    return paletaAvatares[hash.abs() % paletaAvatares.length];
  }

  static ThemeData get light => ThemeData(
    useMaterial3: true,
    colorScheme: ColorScheme.fromSeed(seedColor: azulPrimario),
    scaffoldBackgroundColor: const Color(0xFFF3F6F8),
    appBarTheme: const AppBarTheme(
      backgroundColor: const Color(0xFF080D12),
      foregroundColor: Colors.white,
      centerTitle: false,
      elevation: 0,
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: const Color(0xFFF7F8FA),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(12),
        borderSide: BorderSide.none,
      ),
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: azulSecundario,
        padding: const EdgeInsets.symmetric(vertical: 14),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
        textStyle: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
      ),
    ),
  );

  static ThemeData get dark => ThemeData(
    useMaterial3: true,
    colorScheme: ColorScheme.fromSeed(
      seedColor: azulPrimario,
      brightness: Brightness.dark,
    ),
    scaffoldBackgroundColor: const Color(0xFF080D12),
    appBarTheme: const AppBarTheme(
      backgroundColor: Color(0xFF101B23),
      foregroundColor: Colors.white,
      centerTitle: false,
      elevation: 0,
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: const Color(0xFF16252F),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(12),
        borderSide: BorderSide.none,
      ),
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: const Color(0xFF080D12),
        padding: const EdgeInsets.symmetric(vertical: 14),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
        textStyle: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
      ),
    ),
  );
}
