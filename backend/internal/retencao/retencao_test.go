package retencao

import (
	"context"
	"testing"
	"time"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage/testutil"
)

func TestLimparRespeitaOsPrazos(t *testing.T) {
	pool := testutil.AbrirPool(t)
	ctx := context.Background()
	agora := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)

	empresa := testutil.CriarEmpresa(t, pool)
	motorista := testutil.CriarUsuario(t, pool, empresa, "motorista")
	const orbitID = "ORBIT-ABC123"
	if _, err := pool.Exec(ctx, `insert into orbit (id) values ($1) on conflict do nothing`, orbitID); err != nil {
		t.Fatal(err)
	}
	inserirLeitura := func(idade time.Duration, ponto string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			insert into leitura_orbit (orbit_id, origem, ponto_id, latitude, longitude, detectado_em)
			values ($1, 'celular', $2, -23.5, -46.6, $3)
		`, orbitID, ponto, agora.Add(-idade)); err != nil {
			t.Fatal(err)
		}
	}
	inserirPosicao := func(idade time.Duration) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			insert into posicao (usuario_id, latitude, longitude, capturado_em) values ($1, -23.5, -46.6, $2)
		`, motorista, agora.Add(-idade)); err != nil {
			t.Fatal(err)
		}
	}
	marca := motorista[:8]
	inserirLeitura(91*24*time.Hour, "velho-"+marca)
	inserirLeitura(89*24*time.Hour, "novo-"+marca)
	inserirPosicao(91 * 24 * time.Hour)
	inserirPosicao(89 * 24 * time.Hour)

	if _, err := Limpar(ctx, pool, agora); err != nil {
		t.Fatal(err)
	}

	var leituras, posicoes int
	_ = pool.QueryRow(ctx, `select count(*) from leitura_orbit where ponto_id like '%' || $1`, marca).Scan(&leituras)
	_ = pool.QueryRow(ctx, `select count(*) from posicao where usuario_id = $1`, motorista).Scan(&posicoes)
	if leituras != 1 || posicoes != 1 {
		t.Fatalf("deveria sobrar só o que tem menos de 90 dias: leituras=%d posicoes=%d", leituras, posicoes)
	}
	var restante string
	_ = pool.QueryRow(ctx, `select ponto_id from leitura_orbit where ponto_id like '%' || $1`, marca).Scan(&restante)
	if restante != "novo-"+marca {
		t.Fatalf("sobrou a leitura errada: %s", restante)
	}
}
