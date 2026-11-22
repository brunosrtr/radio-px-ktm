import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/sessao_app.dart';
import '../../core/theme.dart';
import '../../core/rede_local.dart';
import 'auth_service.dart';
import 'conexao_local.dart';

/// Tela de login. Contas do público entram com e-mail; contas de empresa
/// (001) continuam entrando com CPF ou login. Leva ao cadastro aberto e à
/// recuperação de senha (003-rede-orbit, FR-023/FR-025).
class LoginPage extends StatefulWidget {
  const LoginPage({super.key});

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final _formKey = GlobalKey<FormState>();
  final _loginController = TextEditingController();
  final _senhaController = TextEditingController();
  final _authService = AuthService();

  bool _carregando = false;
  bool _mostrarSenha = false;
  String? _erro;

  Future<void> _entrar() async {
    if (_carregando || !_formKey.currentState!.validate()) return;

    setState(() {
      _carregando = true;
      _erro = null;
    });

    try {
      await _authService.login(
        _loginController.text.trim(),
        _senhaController.text,
      );
      if (!mounted) return;
      SessaoApp.instance.entrou();
      context.go('/canais');
    } on ErroConta catch (e) {
      if (!mounted) return;
      setState(() => _erro = e.mensagem);
    } finally {
      if (mounted) setState(() => _carregando = false);
    }
  }

  Future<void> _recuperarSenha() async {
    final email = await context.push<String>('/recuperar-senha');
    if (email != null && mounted) {
      _loginController.text = email;
      _senhaController.clear();
    }
  }

  @override
  void dispose() {
    _loginController.dispose();
    _senhaController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    final escuro = tema.brightness == Brightness.dark;
    return Scaffold(
      backgroundColor: const Color(0xFF080D12),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 24),
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 470),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Image.asset(
                        'assets/branding/logo-ktm.png',
                        width: 126,
                        semanticLabel: 'KTM Indústria',
                      ),
                      IconButton(
                        onPressed: () =>
                            context.push('/sobre'),
                        tooltip: 'Sobre a KTM',
                        icon: const Icon(
                          Icons.info_outline,
                          color: Color(0xFF9BB4C4),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 32),
                  const Text(
                    'KTM · RÁDIO PX',
                    style: TextStyle(
                      color: AppTheme.azulPrimario,
                      fontSize: 11,
                      letterSpacing: 2,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  const SizedBox(height: 14),
                  const Text.rich(
                    TextSpan(
                      children: [
                        TextSpan(
                          text: 'Sua equipe.\n',
                          style: TextStyle(color: Colors.white),
                        ),
                        TextSpan(
                          text: 'Sempre conectada.',
                          style: TextStyle(color: AppTheme.azulPrimario),
                        ),
                      ],
                    ),
                    style: TextStyle(
                      fontSize: 34,
                      fontWeight: FontWeight.w700,
                      letterSpacing: -1.2,
                      height: 1.15,
                    ),
                  ),
                  const SizedBox(height: 14),
                  const Text(
                    'Entre no canal, converse com a equipe e mantenha a central por perto.',
                    style: TextStyle(
                      color: Color(0xFFA9BDCB),
                      fontSize: 14,
                      height: 1.6,
                    ),
                  ),
                  const SizedBox(height: 28),
                  Container(
                    padding: const EdgeInsets.all(24),
                    decoration: BoxDecoration(
                      color: escuro ? const Color(0xFF111E27) : Colors.white,
                      borderRadius: BorderRadius.circular(16),
                      border: Border.all(
                        color: escuro
                            ? const Color(0xFF263B48)
                            : const Color(0xFFE0E8ED),
                      ),
                    ),
                    child: AutofillGroup(
                      child: Form(
                        key: _formKey,
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.stretch,
                          children: [
                            const Text(
                              'ACESSE SUA CONTA',
                              style: TextStyle(
                                color: AppTheme.azulSecundario,
                                fontSize: 10,
                                fontWeight: FontWeight.w700,
                                letterSpacing: 1.5,
                              ),
                            ),
                            const SizedBox(height: 12),
                            Text(
                              'Bem-vindo à equipe.',
                              style: tema.textTheme.titleLarge?.copyWith(
                                fontWeight: FontWeight.w700,
                                letterSpacing: -.6,
                              ),
                            ),
                            const SizedBox(height: 8),
                            Text(
                              'Entre com seu e-mail. Motoristas de empresa usam o CPF ou login de sempre.',
                              style: tema.textTheme.bodySmall,
                            ),
                            const SizedBox(height: 22),
                            if (!RedeLocal.temUrlFixa) ...[
                              Container(
                                padding: const EdgeInsets.all(14),
                                decoration: BoxDecoration(
                                  color: escuro
                                      ? const Color(0xFF0D2631)
                                      : const Color(0xFFF0F9FD),
                                  borderRadius: BorderRadius.circular(10),
                                ),
                                child: const ConexaoLocal(),
                              ),
                              const SizedBox(height: 20),
                            ],
                            TextFormField(
                              controller: _loginController,
                              autofillHints: const [AutofillHints.email, AutofillHints.username],
                              keyboardType: TextInputType.emailAddress,
                              autocorrect: false,
                              decoration: const InputDecoration(
                                labelText: 'E-mail ou login',
                                prefixIcon: Icon(Icons.person_outline),
                              ),
                              textInputAction: TextInputAction.next,
                              validator: (v) => (v == null || v.trim().isEmpty)
                                  ? 'Informe seu e-mail ou login'
                                  : null,
                            ),
                            const SizedBox(height: 16),
                            TextFormField(
                              controller: _senhaController,
                              autofillHints: const [AutofillHints.password],
                              obscureText: !_mostrarSenha,
                              decoration: InputDecoration(
                                labelText: 'Senha',
                                prefixIcon: const Icon(Icons.lock_outline),
                                suffixIcon: IconButton(
                                  onPressed: () => setState(
                                    () => _mostrarSenha = !_mostrarSenha,
                                  ),
                                  tooltip: _mostrarSenha
                                      ? 'Ocultar senha'
                                      : 'Mostrar senha',
                                  icon: Icon(
                                    _mostrarSenha
                                        ? Icons.visibility_off_outlined
                                        : Icons.visibility_outlined,
                                  ),
                                ),
                              ),
                              onFieldSubmitted: (_) => _entrar(),
                              validator: (v) => (v == null || v.isEmpty)
                                  ? 'Informe a senha'
                                  : null,
                            ),
                            Align(
                              alignment: Alignment.centerRight,
                              child: TextButton(
                                onPressed: _carregando ? null : _recuperarSenha,
                                child: const Text('Esqueci minha senha'),
                              ),
                            ),
                            if (_erro != null) ...[
                              const SizedBox(height: 8),
                              Text(
                                _erro!,
                                style: TextStyle(
                                  color: tema.colorScheme.error,
                                  fontSize: 13,
                                ),
                              ),
                            ],
                            const SizedBox(height: 24),
                            FilledButton(
                              style: FilledButton.styleFrom(
                                backgroundColor: const Color(0xFF080D12),
                                foregroundColor: Colors.white,
                                minimumSize: const Size.fromHeight(52),
                              ),
                              onPressed: _carregando ? null : _entrar,
                              child: _carregando
                                  ? const SizedBox(
                                      width: 20,
                                      height: 20,
                                      child: CircularProgressIndicator(
                                        strokeWidth: 2,
                                        color: AppTheme.azulPrimario,
                                      ),
                                    )
                                  : const Row(
                                      mainAxisAlignment:
                                          MainAxisAlignment.spaceBetween,
                                      children: [
                                        Flexible(
                                          child: Text(
                                            'Entrar no Rádio PX',
                                            overflow: TextOverflow.ellipsis,
                                          ),
                                        ),
                                        SizedBox(width: 8),
                                        Icon(
                                          Icons.arrow_forward,
                                          color: AppTheme.azulPrimario,
                                          size: 20,
                                        ),
                                      ],
                                    ),
                            ),
                            const SizedBox(height: 12),
                            OutlinedButton(
                              style: OutlinedButton.styleFrom(
                                minimumSize: const Size.fromHeight(52),
                              ),
                              onPressed: _carregando
                                  ? null
                                  : () => context.go('/cadastro'),
                              child: const Text('Criar conta grátis'),
                            ),
                          ],
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(height: 18),
                  Center(
                    child: TextButton(
                      onPressed: () =>
                          context.push('/sobre'),
                      child: const Text(
                        'Conheça a KTM Indústria',
                        style: TextStyle(
                          color: Color(0xFFAAC7D7),
                          fontSize: 12,
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
