import 'package:flutter/material.dart';

import '../../core/routes.dart';
import '../../core/token_storage.dart';
import '../auth/auth_service.dart';
import '../auth/documentos_legais.dart';
import '../../orbit/ponto_transmissao.dart';
import '../ponto/cartao_ponto_transmissao.dart';
import '../ponto/orbit_virtual_page.dart';

/// Tela provisória "Minha conta": apelido, e-mail, sair e excluir conta
/// (FR-026). Vira a aba Perfil na Etapa 5 da 003-rede-orbit.
class ContaPage extends StatefulWidget {
  const ContaPage({super.key});

  @override
  State<ContaPage> createState() => _ContaPageState();
}

class _ContaPageState extends State<ContaPage> {
  final _authService = AuthService();
  bool _excluindo = false;

  void _irParaLogin() {
    Navigator.of(context).pushNamedAndRemoveUntil(AppRoutes.login, (_) => false);
  }

  Future<void> _confirmarExclusao() async {
    final confirmou = await showDialog<bool>(
      context: context,
      builder: (contexto) => AlertDialog(
        title: const Text('Excluir sua conta?'),
        content: const Text(
          'Seu nome, apelido, e-mail e localização serão apagados. '
          'Isso não pode ser desfeito.',
        ),
        actions: [
          TextButton(onPressed: () => Navigator.of(contexto).pop(false), child: const Text('Cancelar')),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Theme.of(contexto).colorScheme.error),
            onPressed: () => Navigator.of(contexto).pop(true),
            child: const Text('Excluir conta'),
          ),
        ],
      ),
    );
    if (confirmou != true || !mounted) return;

    setState(() => _excluindo = true);
    try {
      await PontoTransmissao.instance.encerrarSessao();
      await _authService.excluirConta();
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Sua conta foi excluída.')));
      _irParaLogin();
    } on ErroConta catch (e) {
      if (!mounted) return;
      setState(() => _excluindo = false);
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(e.mensagem)));
    }
  }

  @override
  Widget build(BuildContext context) {
    final usuario = TokenStorage.instance.usuario;
    return Scaffold(
      appBar: AppBar(title: const Text('Minha conta')),
      body: ListView(
        padding: const EdgeInsets.symmetric(vertical: 8),
        children: [
          ListTile(
            leading: const Icon(Icons.record_voice_over_outlined),
            title: Text(usuario?.apelido ?? '—'),
            subtitle: const Text('Apelido de rádio'),
          ),
          if (usuario?.email != null)
            ListTile(
              leading: const Icon(Icons.alternate_email),
              title: Text(usuario!.email!),
              subtitle: const Text('E-mail'),
            ),
          const Divider(),
          ListenableBuilder(
            listenable: PontoTransmissao.instance,
            builder: (context, _) {
              final ponto = PontoTransmissao.instance;
              return Column(
                children: [
                  SwitchListTile(
                    secondary: const Icon(Icons.sensors),
                    title: const Text('Ponto de transmissão'),
                    subtitle: const Text('Avisa a KTM quando você passa perto de um caminhão com Orbit.'),
                    value: ponto.ligado,
                    onChanged: (ligar) => ligar ? confirmarPontoTransmissao(context) : ponto.desativar(),
                  ),
                  SwitchListTile(
                    secondary: const Icon(Icons.volume_up_outlined),
                    title: const Text('Som do bip'),
                    value: ponto.bipSom,
                    onChanged: (v) => ponto.definirBip(som: v),
                  ),
                  SwitchListTile(
                    secondary: const Icon(Icons.vibration),
                    title: const Text('Vibração do bip'),
                    value: ponto.bipVibracao,
                    onChanged: (v) => ponto.definirBip(vibracao: v),
                  ),
                ],
              );
            },
          ),
          if (PontoTransmissao.podeUsarOrbitVirtual(usuario?.papel))
            ListTile(
              leading: const Icon(Icons.local_shipping_outlined),
              title: const Text('Modo Orbit virtual'),
              subtitle: const Text('Demonstração: este celular se passa por um Orbit.'),
              trailing: const Icon(Icons.chevron_right),
              onTap: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => const OrbitVirtualPage())),
            ),
          const Divider(),
          ListTile(
            leading: const Icon(Icons.description_outlined),
            title: const Text('Termos de uso'),
            onTap: () => abrirDocumentoLegal(context, '/termos'),
          ),
          ListTile(
            leading: const Icon(Icons.privacy_tip_outlined),
            title: const Text('Política de privacidade'),
            onTap: () => abrirDocumentoLegal(context, '/privacidade'),
          ),
          const Divider(),
          ListTile(
            leading: const Icon(Icons.logout),
            title: const Text('Sair'),
            onTap: () async {
              await PontoTransmissao.instance.encerrarSessao();
              _authService.sair();
              _irParaLogin();
            },
          ),
          ListTile(
            key: const Key('excluir-conta'),
            leading: Icon(Icons.delete_forever_outlined, color: Theme.of(context).colorScheme.error),
            title: Text('Excluir conta', style: TextStyle(color: Theme.of(context).colorScheme.error)),
            enabled: !_excluindo,
            onTap: _confirmarExclusao,
          ),
        ],
      ),
    );
  }
}
