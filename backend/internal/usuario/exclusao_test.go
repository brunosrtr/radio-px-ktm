package usuario

import (
	"context"
	"errors"
	"testing"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage/testutil"
)

func TestExcluirConta(t *testing.T) {
	pool := testutil.AbrirPool(t)
	ctx := context.Background()
	repo := NovoRepositorio(pool)

	d := dadosValidos()
	u, err := repo.CriarUsuarioPublico(ctx, d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		insert into posicao_atual (usuario_id, latitude, longitude, capturado_em) values ($1, -23.5, -46.6, now())
	`, u.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.RegistrarDispositivo(ctx, u.ID, "android", "1.0.0", nil); err != nil {
		t.Fatal(err)
	}

	if err := repo.ExcluirConta(ctx, u.ID); err != nil {
		t.Fatalf("erro ao excluir conta: %v", err)
	}

	if _, err := repo.BuscarPorIdentificador(ctx, d.Email); !errors.Is(err, ErrNaoEncontrado) {
		t.Fatalf("login pelo e-mail antigo deveria falhar, veio %v", err)
	}
	if _, err := repo.ObterPorID(ctx, u.ID); !errors.Is(err, ErrNaoEncontrado) {
		t.Fatalf("conta excluída não deveria ser obtida pelo token antigo, veio %v", err)
	}

	var nome string
	var email, apelido *string
	if err := pool.QueryRow(ctx, `select nome, email::text, apelido from usuario where id = $1`, u.ID).Scan(&nome, &email, &apelido); err != nil {
		t.Fatal(err)
	}
	if nome == d.Nome || email != nil || apelido != nil {
		t.Fatalf("dados pessoais deveriam estar anonimizados: nome=%q email=%v apelido=%v", nome, email, apelido)
	}

	for _, tabela := range []string{"posicao_atual", "dispositivo"} {
		var total int
		if err := pool.QueryRow(ctx, `select count(*) from `+tabela+` where usuario_id = $1`, u.ID).Scan(&total); err != nil {
			t.Fatal(err)
		}
		if total != 0 {
			t.Errorf("%s deveria estar vazia para a conta excluída, tem %d", tabela, total)
		}
	}

	if _, err := repo.CriarUsuarioPublico(ctx, d); err != nil {
		t.Fatalf("o mesmo e-mail deveria poder criar uma conta nova, veio %v", err)
	}
	if err := repo.ExcluirConta(ctx, u.ID); !errors.Is(err, ErrNaoEncontrado) {
		t.Fatalf("excluir de novo deveria dar ErrNaoEncontrado, veio %v", err)
	}
}
