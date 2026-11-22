import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/sessao_app.dart';
import 'auth_service.dart';
import 'documentos_legais.dart';
import 'validacoes.dart';

/// Cadastro aberto ao público (FR-023): qualquer pessoa cria a própria
/// conta, sem depender de uma empresa.
class CadastroPage extends StatefulWidget {
  const CadastroPage({super.key, this.authService});

  /// Permite injetar o serviço nos testes de widget.
  final AuthService? authService;

  @override
  State<CadastroPage> createState() => _CadastroPageState();
}

class _CadastroPageState extends State<CadastroPage> {
  final _formKey = GlobalKey<FormState>();
  final _nome = TextEditingController();
  final _apelido = TextEditingController();
  final _email = TextEditingController();
  final _senha = TextEditingController();
  final _confirmacao = TextEditingController();
  late final AuthService _authService = widget.authService ?? AuthService();

  bool _aceitouTermos = false;
  bool _mostrarSenha = false;
  bool _carregando = false;
  String? _erro;

  @override
  void dispose() {
    for (final c in [_nome, _apelido, _email, _senha, _confirmacao]) {
      c.dispose();
    }
    super.dispose();
  }

  Future<void> _criarConta() async {
    if (_carregando || !_aceitouTermos || !_formKey.currentState!.validate()) return;
    setState(() {
      _carregando = true;
      _erro = null;
    });
    try {
      await _authService.cadastrar(
        nome: _nome.text.trim(),
        apelido: _apelido.text.trim(),
        email: _email.text.trim(),
        senha: _senha.text,
      );
      if (!mounted) return;
      SessaoApp.instance.entrou();
      context.go('/canais');
    } on ErroConta catch (e) {
      if (mounted) setState(() => _erro = e.mensagem);
    } finally {
      if (mounted) setState(() => _carregando = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    final estiloLink = TextStyle(
      color: tema.colorScheme.primary,
      fontWeight: FontWeight.w600,
      decoration: TextDecoration.underline,
    );

    return Scaffold(
      appBar: AppBar(title: const Text('Criar conta')),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(20),
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 470),
              child: AutofillGroup(
                child: Form(
                  key: _formKey,
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      Text(
                        'Entre para a rede da estrada.',
                        style: tema.textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.w700),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        'Converse por rádio com quem está no mesmo caminho. Leva menos de um minuto.',
                        style: tema.textTheme.bodyMedium,
                      ),
                      const SizedBox(height: 24),
                      TextFormField(
                        controller: _nome,
                        textCapitalization: TextCapitalization.words,
                        autofillHints: const [AutofillHints.name],
                        textInputAction: TextInputAction.next,
                        decoration: const InputDecoration(
                          labelText: 'Nome completo',
                          prefixIcon: Icon(Icons.badge_outlined),
                        ),
                        validator: validarNome,
                      ),
                      const SizedBox(height: 16),
                      TextFormField(
                        controller: _apelido,
                        textCapitalization: TextCapitalization.words,
                        autofillHints: const [AutofillHints.nickname],
                        textInputAction: TextInputAction.next,
                        maxLength: tamanhoMaximoApelido,
                        decoration: const InputDecoration(
                          labelText: 'Apelido de rádio',
                          helperText: 'É assim que os outros vão ver você falando.',
                          prefixIcon: Icon(Icons.record_voice_over_outlined),
                        ),
                        validator: validarApelido,
                      ),
                      const SizedBox(height: 8),
                      TextFormField(
                        controller: _email,
                        keyboardType: TextInputType.emailAddress,
                        autocorrect: false,
                        autofillHints: const [AutofillHints.email],
                        textInputAction: TextInputAction.next,
                        decoration: const InputDecoration(
                          labelText: 'E-mail',
                          prefixIcon: Icon(Icons.alternate_email),
                        ),
                        validator: validarEmail,
                      ),
                      const SizedBox(height: 16),
                      TextFormField(
                        controller: _senha,
                        obscureText: !_mostrarSenha,
                        autofillHints: const [AutofillHints.newPassword],
                        textInputAction: TextInputAction.next,
                        decoration: InputDecoration(
                          labelText: 'Senha',
                          helperText: 'Pelo menos $tamanhoMinimoSenha caracteres.',
                          prefixIcon: const Icon(Icons.lock_outline),
                          suffixIcon: IconButton(
                            tooltip: _mostrarSenha ? 'Ocultar senha' : 'Mostrar senha',
                            onPressed: () => setState(() => _mostrarSenha = !_mostrarSenha),
                            icon: Icon(_mostrarSenha ? Icons.visibility_off_outlined : Icons.visibility_outlined),
                          ),
                        ),
                        validator: validarSenha,
                      ),
                      const SizedBox(height: 16),
                      TextFormField(
                        controller: _confirmacao,
                        obscureText: !_mostrarSenha,
                        autofillHints: const [AutofillHints.newPassword],
                        decoration: const InputDecoration(
                          labelText: 'Confirme a senha',
                          prefixIcon: Icon(Icons.lock_outline),
                        ),
                        onFieldSubmitted: (_) => _criarConta(),
                        validator: (v) => validarConfirmacao(v, _senha.text),
                      ),
                      const SizedBox(height: 16),
                      CheckboxListTile(
                        key: const Key('aceite-termos'),
                        value: _aceitouTermos,
                        onChanged: (v) => setState(() => _aceitouTermos = v ?? false),
                        controlAffinity: ListTileControlAffinity.leading,
                        contentPadding: EdgeInsets.zero,
                        title: Text.rich(
                          TextSpan(
                            style: tema.textTheme.bodyMedium,
                            children: [
                              const TextSpan(text: 'Li e aceito os '),
                              TextSpan(
                                text: 'termos de uso',
                                style: estiloLink,
                                recognizer: TapGestureRecognizer()
                                  ..onTap = () => abrirDocumentoLegal(context, '/termos'),
                              ),
                              const TextSpan(text: ' e a '),
                              TextSpan(
                                text: 'política de privacidade',
                                style: estiloLink,
                                recognizer: TapGestureRecognizer()
                                  ..onTap = () => abrirDocumentoLegal(context, '/privacidade'),
                              ),
                              const TextSpan(text: '.'),
                            ],
                          ),
                        ),
                      ),
                      if (_erro != null) ...[
                        const SizedBox(height: 8),
                        Text(_erro!, style: TextStyle(color: tema.colorScheme.error)),
                      ],
                      const SizedBox(height: 20),
                      FilledButton(
                        key: const Key('botao-criar-conta'),
                        style: FilledButton.styleFrom(minimumSize: const Size.fromHeight(52)),
                        onPressed: _carregando || !_aceitouTermos ? null : _criarConta,
                        child: _carregando
                            ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2))
                            : const Text('Criar conta'),
                      ),
                      const SizedBox(height: 12),
                      TextButton(
                        onPressed: () => context.go('/entrar'),
                        child: const Text('Já tenho conta'),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
