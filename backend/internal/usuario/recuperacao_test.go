package usuario

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/email"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage/testutil"
)

var padraoCodigo = regexp.MustCompile(`\b\d{6}\b`)

func codigoEnviado(t *testing.T, enviador *email.EnviadorMemoria, para string) string {
	t.Helper()
	m, ok := enviador.Ultima(para)
	if !ok {
		t.Fatalf("nenhum e-mail enviado para %s", para)
	}
	codigo := padraoCodigo.FindString(m.Corpo)
	if codigo == "" {
		t.Fatalf("e-mail sem código de 6 dígitos: %q", m.Corpo)
	}
	return codigo
}

func codigoErrado(certo string) string {
	if certo == "000000" {
		return "111111"
	}
	return "000000"
}

func TestRecuperacaoDeSenha(t *testing.T) {
	pool := testutil.AbrirPool(t)
	ctx := context.Background()
	repo := NovoRepositorio(pool)

	novaConta := func(t *testing.T) *Usuario {
		t.Helper()
		u, err := repo.CriarUsuarioPublico(ctx, dadosValidos())
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	senhaAtual := func(t *testing.T, id string) string {
		t.Helper()
		var hash string
		if err := pool.QueryRow(ctx, `select senha_hash from usuario where id = $1`, id).Scan(&hash); err != nil {
			t.Fatal(err)
		}
		return hash
	}

	t.Run("código certo troca a senha uma única vez", func(t *testing.T) {
		u := novaConta(t)
		enviador := &email.EnviadorMemoria{}
		if err := repo.SolicitarRecuperacao(ctx, u.Email, enviador); err != nil {
			t.Fatal(err)
		}
		codigo := codigoEnviado(t, enviador, u.Email)

		if err := repo.RedefinirSenha(ctx, u.Email, codigo, "nova-senha-123"); err != nil {
			t.Fatalf("esperava redefinir, veio %v", err)
		}
		if !auth.VerificarSenha(senhaAtual(t, u.ID), "nova-senha-123") {
			t.Fatal("a senha não foi trocada")
		}
		if err := repo.RedefinirSenha(ctx, u.Email, codigo, "outra-senha-123"); !errors.Is(err, ErrCodigoInvalido) {
			t.Fatalf("código já usado deveria ser recusado, veio %v", err)
		}
	})

	t.Run("código expirado é recusado", func(t *testing.T) {
		u := novaConta(t)
		enviador := &email.EnviadorMemoria{}
		_ = repo.SolicitarRecuperacao(ctx, u.Email, enviador)
		codigo := codigoEnviado(t, enviador, u.Email)
		if _, err := pool.Exec(ctx, `update recuperacao_senha set expira_em = now() - interval '1 second' where usuario_id = $1`, u.ID); err != nil {
			t.Fatal(err)
		}
		if err := repo.RedefinirSenha(ctx, u.Email, codigo, "nova-senha-123"); !errors.Is(err, ErrCodigoInvalido) {
			t.Fatalf("esperava ErrCodigoInvalido, veio %v", err)
		}
	})

	t.Run("cinco erros invalidam o código", func(t *testing.T) {
		u := novaConta(t)
		enviador := &email.EnviadorMemoria{}
		_ = repo.SolicitarRecuperacao(ctx, u.Email, enviador)
		codigo := codigoEnviado(t, enviador, u.Email)
		for i := 0; i < MaximoTentativasCodigo; i++ {
			if err := repo.RedefinirSenha(ctx, u.Email, codigoErrado(codigo), "nova-senha-123"); !errors.Is(err, ErrCodigoInvalido) {
				t.Fatalf("tentativa %d: esperava ErrCodigoInvalido, veio %v", i+1, err)
			}
		}
		if err := repo.RedefinirSenha(ctx, u.Email, codigo, "nova-senha-123"); !errors.Is(err, ErrCodigoInvalido) {
			t.Fatalf("o código certo depois de 5 erros deveria ser recusado, veio %v", err)
		}
	})

	t.Run("novo pedido invalida o código anterior", func(t *testing.T) {
		u := novaConta(t)
		enviador := &email.EnviadorMemoria{}
		_ = repo.SolicitarRecuperacao(ctx, u.Email, enviador)
		primeiro := codigoEnviado(t, enviador, u.Email)
		_ = repo.SolicitarRecuperacao(ctx, u.Email, enviador)
		segundo := codigoEnviado(t, enviador, u.Email)
		if primeiro != segundo {
			if err := repo.RedefinirSenha(ctx, u.Email, primeiro, "nova-senha-123"); !errors.Is(err, ErrCodigoInvalido) {
				t.Fatalf("o primeiro código deveria ter sido invalidado, veio %v", err)
			}
		}
		if err := repo.RedefinirSenha(ctx, u.Email, segundo, "nova-senha-123"); err != nil {
			t.Fatalf("o código mais recente deveria funcionar, veio %v", err)
		}
	})

	t.Run("e-mail sem conta não envia nada nem dá erro", func(t *testing.T) {
		enviador := &email.EnviadorMemoria{}
		if err := repo.SolicitarRecuperacao(ctx, emailAleatorio(), enviador); err != nil {
			t.Fatalf("não deveria dar erro, veio %v", err)
		}
		if err := repo.SolicitarRecuperacao(ctx, "isto não é e-mail", enviador); err != nil {
			t.Fatalf("não deveria dar erro, veio %v", err)
		}
		if enviador.Total() != 0 {
			t.Fatalf("nenhum e-mail deveria ter sido enviado, foram %d", enviador.Total())
		}
	})

	t.Run("nova senha curta é recusada antes de consumir o código", func(t *testing.T) {
		u := novaConta(t)
		enviador := &email.EnviadorMemoria{}
		_ = repo.SolicitarRecuperacao(ctx, u.Email, enviador)
		codigo := codigoEnviado(t, enviador, u.Email)
		if err := repo.RedefinirSenha(ctx, u.Email, codigo, "curta"); !errors.Is(err, ErrDadosInvalidos) {
			t.Fatalf("esperava ErrDadosInvalidos, veio %v", err)
		}
		if err := repo.RedefinirSenha(ctx, u.Email, codigo, "nova-senha-123"); err != nil {
			t.Fatalf("o código deveria continuar válido, veio %v", err)
		}
	})
}
