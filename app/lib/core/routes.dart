/// Nomes de rota centralizados — evita espalhar strings literais pelo app.
/// `CanalAtivoPage` não tem rota nomeada aqui porque exige argumentos
/// obrigatórios (canalId, nomeCanal); é aberta via `Navigator.push` direto a
/// partir de `ListaCanaisPage`.
class AppRoutes {
  AppRoutes._();

  static const String login = '/login';
  static const String canais = '/canais';
}
