/// Validações dos formulários de conta — as mesmas regras do backend
/// (`usuario/cadastro_publico.go`), para o usuário ver o erro antes de
/// enviar. Cada função devolve a mensagem de erro, ou null se estiver ok.
library;

const tamanhoMinimoSenha = 8;
const tamanhoMinimoApelido = 2;
const tamanhoMaximoApelido = 24;

final _formatoEmail = RegExp(r'^[^\s@]+@[^\s@]+\.[^\s@]+$');

String? validarNome(String? valor) {
  final nome = (valor ?? '').trim();
  if (nome.isEmpty) return 'Informe seu nome';
  if (nome.runes.length > 80) return 'Use até 80 caracteres';
  return null;
}

String? validarApelido(String? valor) {
  final apelido = (valor ?? '').trim().replaceAll(RegExp(r'\s+'), ' ');
  final tamanho = apelido.runes.length;
  if (tamanho < tamanhoMinimoApelido || tamanho > tamanhoMaximoApelido) {
    return 'O apelido deve ter de $tamanhoMinimoApelido a $tamanhoMaximoApelido caracteres';
  }
  return null;
}

String? validarEmail(String? valor) {
  final email = (valor ?? '').trim();
  if (email.isEmpty) return 'Informe seu e-mail';
  if (!_formatoEmail.hasMatch(email)) return 'E-mail inválido';
  return null;
}

String? validarSenha(String? valor) {
  final senha = valor ?? '';
  if (senha.runes.length < tamanhoMinimoSenha) {
    return 'A senha deve ter pelo menos $tamanhoMinimoSenha caracteres';
  }
  return null;
}

String? validarConfirmacao(String? valor, String senha) {
  if ((valor ?? '') != senha) return 'As senhas não conferem';
  return null;
}
