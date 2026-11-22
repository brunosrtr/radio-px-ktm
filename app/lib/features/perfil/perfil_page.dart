import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/sessao_app.dart';
import '../../core/theme.dart';
import '../../core/token_storage.dart';
import '../../orbit/ponto_transmissao.dart';
import '../auth/auth_service.dart';
import '../auth/documentos_legais.dart';
import '../auth/validacoes.dart';

/// Aba Perfil: dados da conta, atalhos e saída (FR-026). O modo Orbit
/// virtual só aparece para contas `admin_ktm` e `demo` (FR-013).
class PerfilPage extends StatefulWidget {
  const PerfilPage({super.key});

  @override
  State<PerfilPage> createState() => _PerfilPageState();
}

class _PerfilPageState extends State<PerfilPage> {
  final _authService = AuthService();
  bool _excluindo = false;

  Future<void> _sair() async {
    await PontoTransmissao.instance.encerrarSessao();
    _authService.sair();
    SessaoApp.instance.saiu();
  }

  Future<void> _editar() async {
    final usuario = TokenStorage.instance.usuario;
    final nome = TextEditingController(text: usuario?.nome ?? '');
    final apelido = TextEditingController(text: usuario?.apelido ?? '');
    final form = GlobalKey<FormState>();
    final salvar = await showDialog<bool>(
      context: context,
      builder: (contexto) => AlertDialog(
        title: const Text('Nome e apelido'),
        content: Form(
          key: form,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextFormField(controller: nome, decoration: const InputDecoration(labelText: 'Nome'), validator: validarNome),
              const SizedBox(height: 12),
              TextFormField(
                controller: apelido,
                maxLength: tamanhoMaximoApelido,
                decoration: const InputDecoration(labelText: 'Apelido de rádio'),
                validator: validarApelido,
              ),
            ],
          ),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.of(contexto).pop(false), child: const Text('Cancelar')),
          FilledButton(
            onPressed: () {
              if (form.currentState!.validate()) Navigator.of(contexto).pop(true);
            },
            child: const Text('Salvar'),
          ),
        ],
      ),
    );
    if (salvar != true) return;
    try {
      await _authService.atualizarPerfil(nome: nome.text.trim(), apelido: apelido.text.trim());
      if (mounted) setState(() {});
    } on ErroConta catch (e) {
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(e.mensagem)));
    }
  }

  Future<void> _confirmarExclusao() async {
    final confirmou = await showDialog<bool>(
      context: context,
      builder: (contexto) => AlertDialog(
        title: const Text('Excluir sua conta?'),
        content: const Text('Seu nome, apelido, e-mail e localização serão apagados. Isso não pode ser desfeito.'),
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
    final mensageiro = ScaffoldMessenger.of(context);
    try {
      await PontoTransmissao.instance.encerrarSessao();
      await _authService.excluirConta();
      mensageiro.showSnackBar(const SnackBar(content: Text('Sua conta foi excluída.')));
      SessaoApp.instance.saiu();
    } on ErroConta catch (e) {
      if (!mounted) return;
      setState(() => _excluindo = false);
      mensageiro.showSnackBar(SnackBar(content: Text(e.mensagem)));
    }
  }

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    final usuario = TokenStorage.instance.usuario;
    final apelido = usuario?.apelido ?? '';
    return SafeArea(
      bottom: false,
      child: ListenableBuilder(
        listenable: PontoTransmissao.instance,
        builder: (context, _) => ListView(
          padding: const EdgeInsets.fromLTRB(16, 24, 16, 24),
          children: [
            Row(
              children: [
                CircleAvatar(
                  radius: 32,
                  backgroundColor: AppTheme.corDoAvatar(apelido),
                  child: Text(AppTheme.iniciais(apelido), style: const TextStyle(color: Colors.white, fontSize: 22, fontWeight: FontWeight.w700)),
                ),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(apelido, style: tema.textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.w700)),
                      if (usuario?.email != null)
                        Text(usuario!.email!, style: tema.textTheme.bodyMedium?.copyWith(color: tema.colorScheme.onSurfaceVariant)),
                    ],
                  ),
                ),
              ],
            ),
            const SizedBox(height: 20),
            Card(
              child: Column(
                children: [
                  ListTile(
                    leading: const Icon(Icons.edit_outlined),
                    title: const Text('Editar nome e apelido'),
                    trailing: const Icon(Icons.chevron_right),
                    onTap: _editar,
                  ),
                  const Divider(),
                  ListTile(
                    leading: const Icon(Icons.sensors, color: AppTheme.ambar),
                    title: const Text('Ponto de transmissão e bip'),
                    trailing: Text(
                      PontoTransmissao.instance.estado == EstadoPonto.ativo ? 'Ativo' : 'Desligado',
                      style: TextStyle(
                        color: PontoTransmissao.instance.estado == EstadoPonto.ativo ? AppTheme.ambar : tema.colorScheme.onSurfaceVariant,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    onTap: () => context.go('/contribuicao'),
                  ),
                ],
              ),
            ),
            if (PontoTransmissao.podeUsarOrbitVirtual(usuario?.papel)) ...[
              const SizedBox(height: 12),
              Container(
                decoration: BoxDecoration(
                  color: AppTheme.fundoOrbit(context),
                  border: Border.all(color: AppTheme.bordaOrbit(context)),
                  borderRadius: BorderRadius.circular(16),
                ),
                child: ListTile(
                  leading: const Icon(Icons.local_shipping_outlined, color: AppTheme.ambar),
                  title: const Text('Modo Orbit virtual', style: TextStyle(fontWeight: FontWeight.w700)),
                  subtitle: Text('Demonstração: este celular se passa por um Orbit.', style: TextStyle(color: AppTheme.textoOrbit(context))),
                  trailing: const Icon(Icons.chevron_right, color: AppTheme.ambar),
                  onTap: () => context.push('/orbit-virtual'),
                ),
              ),
            ],
            const SizedBox(height: 12),
            Card(
              child: Column(
                children: [
                  ListTile(
                    leading: const Icon(Icons.description_outlined),
                    title: const Text('Termos de uso'),
                    trailing: const Icon(Icons.open_in_new, size: 18),
                    onTap: () => abrirDocumentoLegal(context, '/termos'),
                  ),
                  const Divider(),
                  ListTile(
                    leading: const Icon(Icons.privacy_tip_outlined),
                    title: const Text('Política de privacidade'),
                    trailing: const Icon(Icons.open_in_new, size: 18),
                    onTap: () => abrirDocumentoLegal(context, '/privacidade'),
                  ),
                  const Divider(),
                  ListTile(
                    leading: const Icon(Icons.info_outline),
                    title: const Text('Sobre a KTM'),
                    trailing: const Icon(Icons.chevron_right),
                    onTap: () => context.push('/sobre'),
                  ),
                  const Divider(),
                  ListTile(leading: const Icon(Icons.logout), title: const Text('Sair'), onTap: _sair),
                ],
              ),
            ),
            const SizedBox(height: 16),
            OutlinedButton(
              key: const Key('excluir-conta'),
              style: OutlinedButton.styleFrom(
                foregroundColor: tema.colorScheme.error,
                side: BorderSide(color: tema.colorScheme.error.withValues(alpha: 0.6)),
              ),
              onPressed: _excluindo ? null : _confirmarExclusao,
              child: const Text('Excluir minha conta'),
            ),
          ],
        ),
      ),
    );
  }
}
