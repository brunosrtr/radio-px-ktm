/// Dados do usuário autenticado que o app precisa durante a sessão.
class UsuarioSessao {
  const UsuarioSessao({
    required this.id,
    required this.apelido,
    required this.papel,
    required this.temEmpresa,
    this.nome,
    this.email,
  });

  factory UsuarioSessao.deJson(Map<String, dynamic> json) => UsuarioSessao(
    id: json['id'] as String,
    apelido: (json['apelido'] as String?) ?? (json['nome'] as String? ?? ''),
    papel: json['papel'] as String,
    temEmpresa: json['tem_empresa'] as bool? ?? false,
    nome: json['nome'] as String?,
    email: json['email'] as String?,
  );

  final String id;
  final String apelido;
  final String papel;

  /// Contas vinculadas a uma empresa têm histórico de posições; as do
  /// público, não (FR-034 da 003-rede-orbit).
  final bool temEmpresa;
  final String? nome;
  final String? email;
}
