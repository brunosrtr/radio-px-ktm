import 'package:flutter/material.dart';

/// Tema do app, conforme o protótipo de UX aprovado na Etapa 4 da
/// 003-rede-orbit: escuro e de alto contraste por padrão (uso na cabine, sob
/// sol), com versão clara que segue o sistema (FR-039). **Azul é a rádio;
/// âmbar é a rede Orbit / ponto de transmissão.** Alvos de toque de no
/// mínimo 48 px e textos que acompanham o tamanho de fonte do celular.
class AppTheme {
  AppTheme._();

  static const azulPrimario = Color(0xFF41C1F0);
  static const azulSecundario = Color(0xFF087FA9);
  static const ambar = Color(0xFFF5A524);
  static const verdeFalando = Color(0xFF5FD39B);
  static const vermelhoFalando = Color(0xFFB33A3A);

  // Superfícies do tema escuro.
  static const fundoEscuro = Color(0xFF080D12);
  static const superficieEscura = Color(0xFF111E27);
  static const bordaEscura = Color(0xFF22333F);
  static const textoFracoEscuro = Color(0xFFA9BDCB);

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

  static String iniciais(String nome) {
    final partes = nome.trim().split(RegExp(r'\s+'));
    final a = partes.isNotEmpty && partes[0].isNotEmpty ? partes[0][0] : '';
    final b = partes.length > 1 && partes[1].isNotEmpty ? partes[1][0] : '';
    return (a + b).toUpperCase();
  }

  /// Fundo e borda dos cartões da rede Orbit, no tom do tema.
  static Color fundoOrbit(BuildContext context) =>
      Theme.of(context).brightness == Brightness.dark ? const Color(0xFF2A1F08) : const Color(0xFFFFF4DE);
  static Color bordaOrbit(BuildContext context) =>
      Theme.of(context).brightness == Brightness.dark ? const Color(0xFF6B4A0E) : const Color(0xFFE8C27A);
  static Color textoOrbit(BuildContext context) =>
      Theme.of(context).brightness == Brightness.dark ? const Color(0xFFFBE3B3) : const Color(0xFF5C4100);

  static ThemeData get light => _tema(
    ColorScheme.fromSeed(seedColor: azulPrimario, primary: azulSecundario),
    fundo: const Color(0xFFF3F6F8),
    superficie: Colors.white,
    borda: const Color(0xFFDCE4EA),
  );

  static ThemeData get dark => _tema(
    ColorScheme.fromSeed(
      seedColor: azulPrimario,
      brightness: Brightness.dark,
      primary: azulPrimario,
      onPrimary: fundoEscuro,
      surface: fundoEscuro,
      onSurface: const Color(0xFFE8F0F5),
      onSurfaceVariant: textoFracoEscuro,
      surfaceContainerHighest: superficieEscura,
      outlineVariant: bordaEscura,
    ),
    fundo: fundoEscuro,
    superficie: superficieEscura,
    borda: bordaEscura,
  );

  static ThemeData _tema(ColorScheme cores, {required Color fundo, required Color superficie, required Color borda}) {
    final escuro = cores.brightness == Brightness.dark;
    final arredondado = RoundedRectangleBorder(borderRadius: BorderRadius.circular(14));
    return ThemeData(
      useMaterial3: true,
      colorScheme: cores,
      scaffoldBackgroundColor: fundo,
      appBarTheme: AppBarTheme(
        backgroundColor: fundo,
        foregroundColor: cores.onSurface,
        surfaceTintColor: Colors.transparent,
        centerTitle: false,
        elevation: 0,
        titleTextStyle: TextStyle(fontSize: 22, fontWeight: FontWeight.w700, color: cores.onSurface),
      ),
      cardTheme: CardThemeData(
        color: superficie,
        elevation: 0,
        margin: EdgeInsets.zero,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
          side: BorderSide(color: borda),
        ),
      ),
      dividerTheme: DividerThemeData(color: borda, space: 1),
      listTileTheme: const ListTileThemeData(minVerticalPadding: 12, contentPadding: EdgeInsets.symmetric(horizontal: 16)),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: superficie,
        border: OutlineInputBorder(borderRadius: BorderRadius.circular(12), borderSide: BorderSide(color: borda)),
        enabledBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(12), borderSide: BorderSide(color: borda)),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: BorderSide(color: cores.primary, width: 2),
        ),
        contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: azulSecundario,
          foregroundColor: Colors.white,
          minimumSize: const Size(64, 52),
          shape: arredondado,
          textStyle: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          minimumSize: const Size(64, 52),
          shape: arredondado,
          side: BorderSide(color: borda),
          textStyle: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
        ),
      ),
      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(
          minimumSize: const Size(48, 48),
          textStyle: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
        ),
      ),
      navigationBarTheme: NavigationBarThemeData(
        backgroundColor: escuro ? const Color(0xFF0D161D) : Colors.white,
        indicatorColor: escuro ? const Color(0xFF0E3546) : const Color(0xFFD7EFF9),
        height: 72,
        labelTextStyle: WidgetStateProperty.resolveWith(
          (estados) => TextStyle(
            fontSize: 12,
            fontWeight: estados.contains(WidgetState.selected) ? FontWeight.w700 : FontWeight.w600,
          ),
        ),
      ),
      switchTheme: SwitchThemeData(
        trackColor: WidgetStateProperty.resolveWith(
          (estados) => estados.contains(WidgetState.selected) ? azulSecundario : null,
        ),
      ),
    );
  }
}
