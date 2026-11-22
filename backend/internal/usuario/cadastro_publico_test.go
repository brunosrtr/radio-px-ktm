package usuario

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage/testutil"
)

func emailAleatorio() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "pessoa-" + hex.EncodeToString(b) + "@exemplo.com"
}

func dadosValidos() DadosCadastroPublico {
	return DadosCadastroPublico{
		Nome:                "Maria Souza",
		Apelido:             "Maria da Serra",
		Email:               emailAleatorio(),
		Senha:               "senha-forte-1",
		AceitouTermosVersao: VersaoTermosAtual,
	}
}

func TestValidarCadastroPublico(t *testing.T) {
	casos := []struct {
		nome   string
		ajuste func(*DadosCadastroPublico)
		valido bool
	}{
		{"dados completos", func(*DadosCadastroPublico) {}, true},
		{"nome vazio", func(d *DadosCadastroPublico) { d.Nome = "   " }, false},
		{"apelido com 1 caractere", func(d *DadosCadastroPublico) { d.Apelido = "M" }, false},
		{"apelido com 25 caracteres", func(d *DadosCadastroPublico) { d.Apelido = strings.Repeat("a", 25) }, false},
		{"apelido com acento no limite de 24", func(d *DadosCadastroPublico) { d.Apelido = strings.Repeat("é", 24) }, true},
		{"e-mail sem arroba", func(d *DadosCadastroPublico) { d.Email = "maria.exemplo.com" }, false},
		{"e-mail sem domínio", func(d *DadosCadastroPublico) { d.Email = "maria@exemplo" }, false},
		{"e-mail com nome de exibição", func(d *DadosCadastroPublico) { d.Email = "Maria <maria@exemplo.com>" }, false},
		{"senha com 7 caracteres", func(d *DadosCadastroPublico) { d.Senha = "1234567" }, false},
		{"termos não aceitos", func(d *DadosCadastroPublico) { d.AceitouTermosVersao = "" }, false},
		{"termos de versão antiga", func(d *DadosCadastroPublico) { d.AceitouTermosVersao = "2020-01" }, false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			d := dadosValidos()
			c.ajuste(&d)
			_, err := ValidarCadastroPublico(d)
			if c.valido && err != nil {
				t.Fatalf("esperava válido, veio %v", err)
			}
			if !c.valido && !errors.Is(err, ErrDadosInvalidos) {
				t.Fatalf("esperava ErrDadosInvalidos, veio %v", err)
			}
		})
	}
}

func TestCriarUsuarioPublico(t *testing.T) {
	pool := testutil.AbrirPool(t)
	ctx := context.Background()
	repo := NovoRepositorio(pool)

	d := dadosValidos()
	d.Email = "  " + strings.ToUpper(d.Email[:1]) + d.Email[1:] + " "
	u, err := repo.CriarUsuarioPublico(ctx, d)
	if err != nil {
		t.Fatalf("erro ao criar conta: %v", err)
	}
	if u.Papel != PapelUsuario || u.TemEmpresa() {
		t.Fatalf("conta do público deveria ter papel usuario e nenhuma empresa: %+v", u)
	}

	t.Run("e-mail duplicado sem diferenciar maiúsculas", func(t *testing.T) {
		outro := dadosValidos()
		outro.Email = strings.ToLower(strings.TrimSpace(d.Email))
		if _, err := repo.CriarUsuarioPublico(ctx, outro); !errors.Is(err, ErrEmailEmUso) {
			t.Fatalf("esperava ErrEmailEmUso, veio %v", err)
		}
	})

	t.Run("aceite dos termos gravado no cadastro", func(t *testing.T) {
		vigentes, err := repo.ConsentimentosVigentes(ctx, u.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(vigentes) != 1 || vigentes[0].Tipo != ConsentimentoTermos || vigentes[0].Versao != VersaoTermosAtual {
			t.Fatalf("consentimentos inesperados: %+v", vigentes)
		}
	})

	t.Run("login pelo e-mail com outra caixa", func(t *testing.T) {
		achado, err := repo.BuscarPorIdentificador(ctx, strings.ToUpper(strings.TrimSpace(d.Email)))
		if err != nil || achado.ID != u.ID {
			t.Fatalf("esperava encontrar a conta pelo e-mail, veio %v / %+v", err, achado)
		}
	})

	t.Run("conta de empresa antiga continua achada pelo login", func(t *testing.T) {
		empresa := testutil.CriarEmpresa(t, pool)
		motorista := testutil.CriarUsuario(t, pool, empresa, PapelMotorista)
		var login string
		if err := pool.QueryRow(ctx, `select login from usuario where id = $1`, motorista).Scan(&login); err != nil {
			t.Fatal(err)
		}
		achado, err := repo.BuscarPorIdentificador(ctx, " "+login+" ")
		if err != nil || achado.ID != motorista || !achado.TemEmpresa() {
			t.Fatalf("esperava o motorista %s pelo login, veio %v / %+v", motorista, err, achado)
		}
	})

	t.Run("ponto de transmissão pode ser aceito e revogado; termos não", func(t *testing.T) {
		if err := repo.DefinirConsentimento(ctx, u.ID, ConsentimentoPontoTransmissao, true); err != nil {
			t.Fatal(err)
		}
		if err := repo.DefinirConsentimento(ctx, u.ID, ConsentimentoPontoTransmissao, false); err != nil {
			t.Fatal(err)
		}
		vigentes, _ := repo.ConsentimentosVigentes(ctx, u.ID)
		for _, c := range vigentes {
			if c.Tipo == ConsentimentoPontoTransmissao {
				t.Fatalf("consentimento revogado não deveria estar vigente")
			}
		}
		if err := repo.DefinirConsentimento(ctx, u.ID, ConsentimentoTermos, false); !errors.Is(err, ErrConsentimentoInvalido) {
			t.Fatalf("revogar os termos deveria ser recusado, veio %v", err)
		}
		if err := repo.DefinirConsentimento(ctx, u.ID, "marketing", true); !errors.Is(err, ErrConsentimentoInvalido) {
			t.Fatalf("tipo desconhecido deveria ser recusado, veio %v", err)
		}
	})
}
