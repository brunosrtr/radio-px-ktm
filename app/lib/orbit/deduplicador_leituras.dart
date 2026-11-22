/// Decide quais detecções viram leitura (FR-003, FR-015): no máximo uma
/// leitura por Orbit a cada [intervaloMinimo], e nunca do Orbit virtual que
/// o próprio aparelho está anunciando. Puro e sem dependência de plataforma
/// — regra crítica testada (Princípio IV).
class DeduplicadorLeituras {
  DeduplicadorLeituras({this.intervaloMinimo = const Duration(seconds: 30)});

  final Duration intervaloMinimo;
  final _ultimaAceita = <String, DateTime>{};

  /// Orbit que este aparelho está anunciando no modo Orbit virtual.
  String? orbitProprio;

  /// Registra a detecção e devolve true se ela deve virar uma leitura.
  bool aceitar(String orbitId, DateTime instante) {
    if (orbitId == orbitProprio) return false;
    final ultima = _ultimaAceita[orbitId];
    if (ultima != null && instante.difference(ultima) < intervaloMinimo) {
      return false;
    }
    _ultimaAceita[orbitId] = instante;
    // Esquece Orbits que já saíram da janela, para o mapa não crescer.
    _ultimaAceita.removeWhere(
      (id, quando) => id != orbitId && instante.difference(quando) >= intervaloMinimo,
    );
    return true;
  }

  void limpar() => _ultimaAceita.clear();
}
