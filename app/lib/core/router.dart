import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../features/auth/cadastro_page.dart';
import '../features/auth/login_page.dart';
import '../features/auth/recuperar_senha_page.dart';
import '../features/auth/sobre_page.dart';
import '../features/canais/canais_page.dart';
import '../features/canais/canal_ativo_page.dart';
import '../features/canais/convite_page.dart';
import '../features/canais/criar_canal_page.dart';
import '../features/canais/entrar_convite.dart';
import '../features/contribuicao/contribuicao_page.dart';
import '../features/explorar/explorar_page.dart';
import '../features/onboarding/boas_vindas_page.dart';
import '../features/onboarding/permissoes_page.dart';
import '../features/perfil/perfil_page.dart';
import '../features/ponto/orbit_virtual_page.dart';
import '../features/shell/casca_page.dart';
import 'sessao_app.dart';

/// Dados que a lista de canais passa ao abrir um canal.
class AberturaCanal {
  const AberturaCanal({required this.nome, this.silenciado = false});

  final String nome;
  final bool silenciado;
}

final _chaveRaiz = GlobalKey<NavigatorState>(debugLabel: 'raiz');

/// Rotas do app (FR-038): 4 abas numa casca com `NavigationBar`, mais as
/// telas de entrada e as de tela cheia (canal ativo, Orbit virtual). A
/// regra de redirecionamento está em [decidirRota].
GoRouter criarRouter({SessaoApp? sessao, String? inicial}) {
  final estado = sessao ?? SessaoApp.instance;
  return GoRouter(
    navigatorKey: _chaveRaiz,
    initialLocation: inicial ?? '/canais',
    refreshListenable: estado,
    redirect: (context, state) => estado.redirecionar(state.matchedLocation),
    routes: [
      GoRoute(path: '/boas-vindas', builder: (_, _) => const BoasVindasPage()),
      GoRoute(path: '/entrar', builder: (_, _) => const LoginPage()),
      GoRoute(path: '/cadastro', builder: (_, _) => const CadastroPage()),
      GoRoute(path: '/recuperar-senha', builder: (_, _) => const RecuperarSenhaPage()),
      GoRoute(path: '/sobre', builder: (_, _) => const SobrePage()),
      GoRoute(path: '/permissoes', builder: (_, _) => const PermissoesPage()),
      GoRoute(
        path: '/canal/:id',
        parentNavigatorKey: _chaveRaiz,
        builder: (_, state) {
          final abertura = state.extra is AberturaCanal ? state.extra! as AberturaCanal : const AberturaCanal(nome: 'Canal');
          return CanalAtivoPage(
            canalId: state.pathParameters['id']!,
            nomeCanal: abertura.nome,
            silenciadoInicial: abertura.silenciado,
          );
        },
      ),
      GoRoute(
        path: '/canal/:id/convite',
        parentNavigatorKey: _chaveRaiz,
        builder: (_, state) => ConvitePage(canalId: state.pathParameters['id']!),
      ),
      GoRoute(path: '/criar-canal', parentNavigatorKey: _chaveRaiz, builder: (_, _) => const CriarCanalPage()),
      // Link do QR: radiopx://app/convite/<código> (FR-030).
      GoRoute(
        path: '/convite/:codigo',
        parentNavigatorKey: _chaveRaiz,
        builder: (_, state) => EntrarPorLinkPage(codigo: state.pathParameters['codigo']!),
      ),
      GoRoute(path: '/orbit-virtual', parentNavigatorKey: _chaveRaiz, builder: (_, _) => const OrbitVirtualPage()),
      StatefulShellRoute.indexedStack(
        builder: (_, _, navegacao) => CascaPage(navegacao: navegacao),
        branches: [
          StatefulShellBranch(routes: [GoRoute(path: '/canais', builder: (_, _) => const CanaisPage())]),
          StatefulShellBranch(routes: [GoRoute(path: '/explorar', builder: (_, _) => const ExplorarPage())]),
          StatefulShellBranch(routes: [GoRoute(path: '/contribuicao', builder: (_, _) => const ContribuicaoPage())]),
          StatefulShellBranch(routes: [GoRoute(path: '/perfil', builder: (_, _) => const PerfilPage())]),
        ],
      ),
    ],
  );
}
