import 'package:flutter/foundation.dart';
import 'package:hive_flutter/hive_flutter.dart';

import '../models/usuario_sessao.dart';
import 'token_storage.dart';

/// Rotas que não exigem conta.
const rotasPublicas = {'/boas-vindas', '/entrar', '/cadastro', '/recuperar-senha', '/sobre'};

/// Rotas que não fazem sentido para quem já entrou.
const rotasDeEntrada = {'/boas-vindas', '/entrar', '/cadastro', '/recuperar-senha'};

/// Para onde o app deve ir antes de abrir [destino], ou null para seguir.
/// Pura e testada: é a regra de navegação de primeiro acesso (FR-035/036).
String? decidirRota({
  required String destino,
  required bool logado,
  required bool apresentacaoVista,
  required bool permissoesVistas,
}) {
  if (!logado) {
    if (rotasPublicas.contains(destino)) return null;
    return apresentacaoVista ? '/entrar' : '/boas-vindas';
  }
  if (!permissoesVistas) {
    return destino == '/permissoes' || destino == '/sobre' ? null : '/permissoes';
  }
  if (rotasDeEntrada.contains(destino) || destino == '/permissoes') return '/canais';
  return null;
}

/// Estado do app que decide a navegação: se há sessão e se a apresentação e
/// as permissões já foram vistas neste aparelho. Avisa o roteador a cada
/// mudança.
class SessaoApp extends ChangeNotifier {
  SessaoApp._();

  static final instance = SessaoApp._();

  Box<dynamic>? _preferencias;
  bool apresentacaoVista = false;
  bool permissoesVistas = false;

  bool get logado => TokenStorage.instance.token != null;

  UsuarioSessao? get usuario => TokenStorage.instance.usuario;

  /// Lê o que já foi visto neste aparelho. Sem Hive (testes), fica tudo como
  /// não visto.
  Future<void> carregar() async {
    _preferencias = await Hive.openBox<dynamic>('preferencias');
    apresentacaoVista = _preferencias!.get('apresentacao_vista', defaultValue: false) as bool;
    permissoesVistas = _preferencias!.get('permissoes_vistas', defaultValue: false) as bool;
  }

  Future<void> marcarApresentacaoVista() async {
    apresentacaoVista = true;
    await _preferencias?.put('apresentacao_vista', true);
    notifyListeners();
  }

  Future<void> marcarPermissoesVistas() async {
    permissoesVistas = true;
    await _preferencias?.put('permissoes_vistas', true);
    notifyListeners();
  }

  /// Chamado depois de login ou cadastro bem-sucedidos.
  void entrou() {
    apresentacaoVista = true;
    _preferencias?.put('apresentacao_vista', true);
    notifyListeners();
  }

  /// Chamado ao sair ou excluir a conta (o token já foi limpo).
  void saiu() => notifyListeners();

  String? redirecionar(String destino) => decidirRota(
    destino: destino,
    logado: logado,
    apresentacaoVista: apresentacaoVista,
    permissoesVistas: permissoesVistas,
  );
}
