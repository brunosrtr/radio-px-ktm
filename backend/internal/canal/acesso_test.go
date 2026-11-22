package canal

import (
	"errors"
	"testing"
)

// Princípio IV: regra única de entrada em canais da comunidade e de empresa.
func TestPodeEntrar(t *testing.T) {
	publico := Acesso{Comunidade: true, Publico: true, Limite: 10}
	privado := Acesso{Comunidade: true, Publico: false, Limite: 10}
	empresa := Acesso{Comunidade: false, Limite: 10}

	com := func(a Acesso, f func(*Acesso)) Acesso { f(&a); return a }

	casos := []struct {
		nome   string
		acesso Acesso
		erro   error
	}{
		{"público entra", publico, nil},
		{"privado sem convite não entra", privado, ErrSemPermissao},
		{"privado com convite entra", com(privado, func(a *Acesso) { a.Situacao = SituacaoLiberado }), nil},
		{"criador entra no privado sem convite", com(privado, func(a *Acesso) { a.EhCriador = true }), nil},
		{"banido não entra nem em canal público", com(publico, func(a *Acesso) { a.Situacao = SituacaoBanido }), ErrBanido},
		{"banido não volta pelo convite antigo", com(privado, func(a *Acesso) { a.Situacao = SituacaoBanido }), ErrBanido},
		{"canal apagado", com(publico, func(a *Acesso) { a.Apagado = true }), ErrNaoEncontrado},
		{"canal de empresa sem autorização", empresa, ErrSemPermissao},
		{"canal de empresa autorizado", com(empresa, func(a *Acesso) { a.EmpresaAutorizada = true }), nil},
		{"conta do público não vira autorizada por ser 'pública'", com(empresa, func(a *Acesso) { a.Publico = true }), ErrSemPermissao},
		{"fora da área", com(publico, func(a *Acesso) { a.GeocercaAtiva = true }), ErrForaDaArea},
		{"dentro da área", com(publico, func(a *Acesso) { a.GeocercaAtiva, a.DentroDaArea = true, true }), nil},
		{"canal cheio", com(publico, func(a *Acesso) { a.Participantes = 10 }), ErrCanalCheio},
		{"última vaga", com(publico, func(a *Acesso) { a.Participantes = 9 }), nil},
		{"criador também respeita o limite", com(privado, func(a *Acesso) { a.EhCriador, a.Participantes = true, 10 }), ErrCanalCheio},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if err := PodeEntrar(c.acesso); !errors.Is(err, c.erro) && !(err == nil && c.erro == nil) {
				t.Fatalf("esperava %v, veio %v", c.erro, err)
			}
		})
	}
}

func TestCodigoConvite(t *testing.T) {
	vistos := map[string]bool{}
	for range 200 {
		c, err := GerarCodigoConvite()
		if err != nil {
			t.Fatal(err)
		}
		if NormalizarCodigoConvite(c) != c {
			t.Fatalf("código gerado não é válido: %q", c)
		}
		vistos[c] = true
	}
	if len(vistos) < 199 {
		t.Fatalf("códigos repetidos demais: %d distintos em 200", len(vistos))
	}

	for entrada, esperado := range map[string]string{
		"K7QM-2XPA":                      "K7QM2XPA",
		" k7qm 2xpa ":                    "K7QM2XPA",
		"radiopx://app/convite/K7QM2XPA": "K7QM2XPA",
		"K7QM-2XP":                       "",
		"K7QM-2XP0":                      "",
		"":                               "",
	} {
		if got := NormalizarCodigoConvite(entrada); got != esperado {
			t.Errorf("NormalizarCodigoConvite(%q) = %q, esperado %q", entrada, got, esperado)
		}
	}
	if FormatarCodigoConvite("K7QM2XPA") != "K7QM-2XPA" {
		t.Error("formatação com hífen")
	}
}
