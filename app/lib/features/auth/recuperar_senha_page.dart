import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'auth_service.dart';
import 'validacoes.dart';

/// Recuperação de senha em dois passos (FR-025): informar o e-mail e depois
/// o código de 6 dígitos recebido, junto com a nova senha.
class RecuperarSenhaPage extends StatefulWidget {
  const RecuperarSenhaPage({super.key, this.authService});

  final AuthService? authService;

  @override
  State<RecuperarSenhaPage> createState() => _RecuperarSenhaPageState();
}

class _RecuperarSenhaPageState extends State<RecuperarSenhaPage> {
  final _formEmail = GlobalKey<FormState>();
  final _formCodigo = GlobalKey<FormState>();
  final _email = TextEditingController();
  final _codigo = TextEditingController();
  final _senha = TextEditingController();
  late final AuthService _authService = widget.authService ?? AuthService();

  bool _codigoEnviado = false;
  bool _carregando = false;
  String? _erro;

  @override
  void dispose() {
    _email.dispose();
    _codigo.dispose();
    _senha.dispose();
    super.dispose();
  }

  Future<void> _executar(Future<void> Function() acao) async {
    if (_carregando) return;
    setState(() {
      _carregando = true;
      _erro = null;
    });
    try {
      await acao();
    } on ErroConta catch (e) {
      if (mounted) setState(() => _erro = e.mensagem);
    } finally {
      if (mounted) setState(() => _carregando = false);
    }
  }

  Future<void> _enviarCodigo() async {
    if (!_formEmail.currentState!.validate()) return;
    await _executar(() async {
      await _authService.solicitarRecuperacao(_email.text.trim());
      if (mounted) setState(() => _codigoEnviado = true);
    });
  }

  Future<void> _redefinir() async {
    if (!_formCodigo.currentState!.validate()) return;
    await _executar(() async {
      await _authService.redefinirSenha(
        email: _email.text.trim(),
        codigo: _codigo.text.trim(),
        novaSenha: _senha.text,
      );
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Senha alterada. Entre com a nova senha.')),
      );
      Navigator.of(context).pop(_email.text.trim());
    });
  }

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('Recuperar senha')),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(20),
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 470),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Form(
                    key: _formEmail,
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        Text(
                          'Informe o e-mail da sua conta. Vamos enviar um código de 6 dígitos.',
                          style: tema.textTheme.bodyLarge,
                        ),
                        const SizedBox(height: 16),
                        TextFormField(
                          controller: _email,
                          enabled: !_codigoEnviado,
                          keyboardType: TextInputType.emailAddress,
                          autocorrect: false,
                          autofillHints: const [AutofillHints.email],
                          decoration: const InputDecoration(
                            labelText: 'E-mail',
                            prefixIcon: Icon(Icons.alternate_email),
                          ),
                          validator: validarEmail,
                          onFieldSubmitted: (_) => _enviarCodigo(),
                        ),
                        const SizedBox(height: 16),
                        if (!_codigoEnviado)
                          FilledButton(
                            style: FilledButton.styleFrom(minimumSize: const Size.fromHeight(52)),
                            onPressed: _carregando ? null : _enviarCodigo,
                            child: const Text('Enviar código'),
                          ),
                      ],
                    ),
                  ),
                  if (_codigoEnviado)
                    Form(
                      key: _formCodigo,
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          Container(
                            padding: const EdgeInsets.all(14),
                            decoration: BoxDecoration(
                              color: tema.colorScheme.primaryContainer,
                              borderRadius: BorderRadius.circular(12),
                            ),
                            child: Text(
                              'Se o e-mail tiver conta, enviamos um código. Ele vale por 15 minutos. '
                              'Confira também a caixa de spam.',
                              style: TextStyle(color: tema.colorScheme.onPrimaryContainer),
                            ),
                          ),
                          const SizedBox(height: 16),
                          TextFormField(
                            controller: _codigo,
                            keyboardType: TextInputType.number,
                            autofillHints: const [AutofillHints.oneTimeCode],
                            inputFormatters: [
                              FilteringTextInputFormatter.digitsOnly,
                              LengthLimitingTextInputFormatter(6),
                            ],
                            decoration: const InputDecoration(
                              labelText: 'Código de 6 dígitos',
                              prefixIcon: Icon(Icons.pin_outlined),
                            ),
                            validator: (v) => (v ?? '').trim().length == 6 ? null : 'Informe os 6 dígitos',
                          ),
                          const SizedBox(height: 16),
                          TextFormField(
                            controller: _senha,
                            obscureText: true,
                            autofillHints: const [AutofillHints.newPassword],
                            decoration: const InputDecoration(
                              labelText: 'Nova senha',
                              prefixIcon: Icon(Icons.lock_outline),
                            ),
                            validator: validarSenha,
                            onFieldSubmitted: (_) => _redefinir(),
                          ),
                          const SizedBox(height: 16),
                          FilledButton(
                            style: FilledButton.styleFrom(minimumSize: const Size.fromHeight(52)),
                            onPressed: _carregando ? null : _redefinir,
                            child: const Text('Trocar senha'),
                          ),
                          TextButton(
                            onPressed: _carregando
                                ? null
                                : () => setState(() {
                                    _codigoEnviado = false;
                                    _codigo.clear();
                                  }),
                            child: const Text('Não recebi / pedir outro código'),
                          ),
                        ],
                      ),
                    ),
                  if (_erro != null) ...[
                    const SizedBox(height: 12),
                    Text(_erro!, style: TextStyle(color: tema.colorScheme.error)),
                  ],
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
