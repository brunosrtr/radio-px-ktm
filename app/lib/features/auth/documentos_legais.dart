import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../core/env.dart';
import '../../core/rede_local.dart';

/// Abre no navegador os termos de uso (`/termos`) ou a política de
/// privacidade (`/privacidade`) servidos pelo backend.
Future<void> abrirDocumentoLegal(BuildContext context, String caminho) async {
  final mensageiro = ScaffoldMessenger.of(context);
  try {
    await RedeLocal.instance.assegurarConexao();
    final aberto = await launchUrl(
      Uri.parse('${Env.apiBaseUrl}$caminho'),
      mode: LaunchMode.externalApplication,
    );
    if (!aberto) throw StateError('não abriu');
  } catch (_) {
    mensageiro.showSnackBar(
      const SnackBar(
        content: Text('Não foi possível abrir o documento agora. Confira a conexão.'),
      ),
    );
  }
}
