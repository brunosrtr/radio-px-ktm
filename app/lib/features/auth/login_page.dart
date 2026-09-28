import 'package:dio/dio.dart';
import 'package:flutter/material.dart';

import '../../core/routes.dart';
import '../../core/theme.dart';
import 'auth_service.dart';

/// Tela de login — contas são fornecidas pela empresa, sem autocadastro
/// (research.md §5).
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
  String? _erro;

  Future<void> _entrar() async {
    if (!_formKey.currentState!.validate()) return;

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
      Navigator.of(context).pushReplacementNamed(AppRoutes.canais);
    } on DioException catch (e) {
      final dados = e.response?.data;
      final mensagem = dados is Map ? dados['mensagem'] as String? : null;
      setState(
        () => _erro = mensagem ?? 'Não foi possível entrar. Verifique login e senha.',
      );
    } finally {
      if (mounted) setState(() => _carregando = false);
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
    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: [AppTheme.azulPrimario, Color(0xFF1C8FC7)],
          ),
        ),
        child: SafeArea(
          child: Center(
            child: SingleChildScrollView(
              padding: const EdgeInsets.all(24),
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 380),
                child: Container(
                  padding: const EdgeInsets.fromLTRB(28, 36, 28, 28),
                  decoration: BoxDecoration(
                    color: Theme.of(context).scaffoldBackgroundColor,
                    borderRadius: BorderRadius.circular(20),
                    boxShadow: [
                      BoxShadow(
                        color: Colors.black.withValues(alpha: 0.25),
                        blurRadius: 40,
                        offset: const Offset(0, 20),
                      ),
                    ],
                  ),
                  child: Form(
                    key: _formKey,
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Container(
                          width: 64,
                          height: 64,
                          decoration: const BoxDecoration(
                            shape: BoxShape.circle,
                            gradient: LinearGradient(
                              begin: Alignment.topLeft,
                              end: Alignment.bottomRight,
                              colors: [
                                AppTheme.azulPrimario,
                                AppTheme.azulSecundario,
                              ],
                            ),
                          ),
                          alignment: Alignment.center,
                          child: const Text(
                            '📻',
                            style: TextStyle(fontSize: 30),
                          ),
                        ),
                        const SizedBox(height: 16),
                        Text(
                          'Rádio PX Digital',
                          style: Theme.of(context).textTheme.headlineSmall
                              ?.copyWith(fontWeight: FontWeight.w700),
                        ),
                        const SizedBox(height: 4),
                        Text(
                          'Entre com a conta da sua empresa',
                          style: Theme.of(context).textTheme.bodySmall,
                        ),
                        const SizedBox(height: 28),
                        TextFormField(
                          controller: _loginController,
                          decoration: const InputDecoration(labelText: 'Login'),
                          textInputAction: TextInputAction.next,
                          validator: (v) =>
                              (v == null || v.isEmpty) ? 'Informe o login' : null,
                        ),
                        const SizedBox(height: 12),
                        TextFormField(
                          controller: _senhaController,
                          decoration: const InputDecoration(labelText: 'Senha'),
                          obscureText: true,
                          onFieldSubmitted: (_) => _entrar(),
                          validator: (v) =>
                              (v == null || v.isEmpty) ? 'Informe a senha' : null,
                        ),
                        if (_erro != null) ...[
                          const SizedBox(height: 12),
                          Text(
                            _erro!,
                            style: const TextStyle(color: Colors.red),
                            textAlign: TextAlign.center,
                          ),
                        ],
                        const SizedBox(height: 24),
                        SizedBox(
                          width: double.infinity,
                          child: FilledButton(
                            onPressed: _carregando ? null : _entrar,
                            child: _carregando
                                ? const SizedBox(
                                    width: 20,
                                    height: 20,
                                    child: CircularProgressIndicator(
                                      strokeWidth: 2,
                                      color: Colors.white,
                                    ),
                                  )
                                : const Text('Entrar'),
                          ),
                        ),
                      ],
                    ),
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
