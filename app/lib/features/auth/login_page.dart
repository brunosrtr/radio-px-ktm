import 'package:dio/dio.dart';
import 'package:flutter/material.dart';

import '../../core/routes.dart';
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
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 360),
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Form(
              key: _formKey,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    'Rádio PX Digital',
                    style: Theme.of(context).textTheme.headlineSmall,
                  ),
                  const SizedBox(height: 24),
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
                    Text(_erro!, style: const TextStyle(color: Colors.red)),
                  ],
                  const SizedBox(height: 24),
                  FilledButton(
                    onPressed: _carregando ? null : _entrar,
                    child: _carregando
                        ? const SizedBox(
                            width: 20,
                            height: 20,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Text('Entrar'),
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
