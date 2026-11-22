package canal

import "errors"

var (
	// ErrBanido indica que o criador removeu o usuário do canal.
	ErrBanido = errors.New("canal: você foi removido deste canal")
	// ErrCanalCheio indica que o canal atingiu o limite de participantes.
	ErrCanalCheio = errors.New("canal: canal cheio")
)

// Situações de um usuário num canal da comunidade (tabela acesso_canal).
const (
	SituacaoNenhuma  = ""
	SituacaoLiberado = "liberado"
	SituacaoBanido   = "banido"
)

// Acesso reúne tudo o que decide se um usuário pode entrar num canal. Quem
// chama preenche a partir do banco, da posição e do hub; a decisão em si é
// pura (Princípio IV, data-model.md §3).
type Acesso struct {
	Apagado bool
	// Canal da comunidade (sem empresa) e sua visibilidade.
	Comunidade bool
	Publico    bool
	EhCriador  bool
	Situacao   string
	// Canal de empresa liberado para a empresa do usuário (regra da 001).
	EmpresaAutorizada bool
	GeocercaAtiva     bool
	DentroDaArea      bool
	Participantes     int
	Limite            int
}

// PodeEntrar aplica, nesta ordem: banido → apagado → sem permissão → fora
// da área → canal cheio. nil significa que pode entrar.
func PodeEntrar(a Acesso) error {
	switch {
	case a.Situacao == SituacaoBanido:
		return ErrBanido
	case a.Apagado:
		return ErrNaoEncontrado
	case !podeVer(a):
		return ErrSemPermissao
	case a.GeocercaAtiva && !a.DentroDaArea:
		return ErrForaDaArea
	case a.Limite > 0 && a.Participantes >= a.Limite:
		return ErrCanalCheio
	}
	return nil
}

func podeVer(a Acesso) bool {
	if a.EhCriador {
		return true
	}
	if a.Comunidade {
		return a.Publico || a.Situacao == SituacaoLiberado
	}
	return a.EmpresaAutorizada
}
