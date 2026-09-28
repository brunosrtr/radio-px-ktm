import 'package:flutter/material.dart';

/// Tema visual único do app — motoristas usam em cabine, muitas vezes sob luz
/// solar direta, daí o contraste alto e os alvos de toque grandes herdados do
/// `ThemeData` padrão do Material 3. Paleta azul, no mesmo espírito visual do
/// painel web da empresa.
class AppTheme {
  AppTheme._();

  static const azulPrimario = Color(0xFF2AABEE);
  static const azulSecundario = Color(0xFF229ED9);

  static const paletaAvatares = [
    Color(0xFFE17076),
    Color(0xFFEDA86C),
    Color(0xFFA695E7),
    Color(0xFF7BC862),
    Color(0xFF6EC9CB),
    Color(0xFF65AADD),
    Color(0xFFEE7AAE),
    Color(0xFFF2777A),
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
    scaffoldBackgroundColor: const Color(0xFFEEF2F5),
    appBarTheme: const AppBarTheme(
      backgroundColor: azulPrimario,
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
    scaffoldBackgroundColor: const Color(0xFF17212B),
    appBarTheme: const AppBarTheme(
      backgroundColor: Color(0xFF1E2C3A),
      foregroundColor: Colors.white,
      centerTitle: false,
      elevation: 0,
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: const Color(0xFF242F3D),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(12),
        borderSide: BorderSide.none,
      ),
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: azulPrimario,
        padding: const EdgeInsets.symmetric(vertical: 14),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
        textStyle: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
      ),
    ),
  );
}
