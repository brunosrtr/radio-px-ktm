package orbit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage/testutil"
)

func orbitAleatorio() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return "ORBIT-" + strings.ToUpper(hex.EncodeToString(b))
}

// Princípio IV: ingestão em lote de leituras, idempotente, com atualização
// condicional da última posição do Orbit.
func TestIngerirLoteIdempotenteEAtualizacaoCondicional(t *testing.T) {
	pool := testutil.AbrirPool(t)
	ctx := context.Background()
	repo := NovoRepositorio(pool)
	usuario := testutil.CriarUsuarioPublico(t, pool)

	id := orbitAleatorio()
	base := time.Now().Add(-time.Hour).UTC().Truncate(time.Millisecond)
	lote := []LeituraRecebida{
		{OrbitID: id, Latitude: -23.50, Longitude: -46.60, DetectadoEm: base},
		{OrbitID: id, Latitude: -23.51, Longitude: -46.61, DetectadoEm: base.Add(30 * time.Second)},
	}

	r1, err := repo.IngerirLote(ctx, OrigemCelular, "ponto-a", usuario, lote)
	if err != nil {
		t.Fatal(err)
	}
	if r1.Aceitas != 2 || r1.Duplicadas != 0 || len(r1.OrbitsAtualizados) != 1 {
		t.Fatalf("primeiro lote: %+v", r1)
	}
	o := r1.OrbitsAtualizados[0]
	if *o.Latitude != -23.51 || !o.UltimaLeituraEm.Equal(base.Add(30*time.Second)) {
		t.Fatalf("a última posição deveria ser a da leitura mais recente do lote: %+v", o)
	}

	t.Run("reenviar o mesmo lote não duplica", func(t *testing.T) {
		r2, err := repo.IngerirLote(ctx, OrigemCelular, "ponto-a", usuario, lote)
		if err != nil {
			t.Fatal(err)
		}
		if r2.Aceitas != 0 || r2.Duplicadas != 2 || len(r2.OrbitsAtualizados) != 0 {
			t.Fatalf("reenvio: %+v", r2)
		}
		var total int
		_ = pool.QueryRow(ctx, `select count(*) from leitura_orbit where orbit_id = $1`, id).Scan(&total)
		if total != 2 {
			t.Fatalf("esperava 2 leituras gravadas, há %d", total)
		}
	})

	t.Run("leitura mais antiga de outro ponto não move o Orbit", func(t *testing.T) {
		antiga := []LeituraRecebida{{OrbitID: id, Latitude: -22.00, Longitude: -45.00, DetectadoEm: base.Add(-10 * time.Minute)}}
		r3, err := repo.IngerirLote(ctx, OrigemCelular, "ponto-b", "", antiga)
		if err != nil {
			t.Fatal(err)
		}
		if r3.Aceitas != 1 || len(r3.OrbitsAtualizados) != 0 {
			t.Fatalf("leitura antiga deveria ser gravada sem mover o Orbit: %+v", r3)
		}
		var lat float64
		_ = pool.QueryRow(ctx, `select ultima_latitude::float8 from orbit where id = $1`, id).Scan(&lat)
		if lat != -23.51 {
			t.Fatalf("o Orbit voltou para uma posição antiga: %v", lat)
		}
	})

	t.Run("leitura mais nova move o Orbit", func(t *testing.T) {
		nova := []LeituraRecebida{{OrbitID: id, Latitude: -23.60, Longitude: -46.70, DetectadoEm: base.Add(5 * time.Minute)}}
		r4, err := repo.IngerirLote(ctx, OrigemCelular, "ponto-b", "", nova)
		if err != nil {
			t.Fatal(err)
		}
		if len(r4.OrbitsAtualizados) != 1 || *r4.OrbitsAtualizados[0].Latitude != -23.60 {
			t.Fatalf("leitura nova deveria mover o Orbit: %+v", r4)
		}
		if r4.OrbitsAtualizados[0].PontosDistintos24h != 2 {
			t.Fatalf("esperava 2 pontos distintos em 24 h, veio %d", r4.OrbitsAtualizados[0].PontosDistintos24h)
		}
	})

	t.Run("contribuição conta só as leituras novas do usuário", func(t *testing.T) {
		hoje, semana, total, err := repo.Contribuicao(ctx, usuario, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if total != 2 || semana != 2 || hoje > 2 {
			t.Fatalf("contribuição esperada: total 2 (o reenvio não conta); veio hoje=%d semana=%d total=%d", hoje, semana, total)
		}
	})

	t.Run("histórico em ordem cronológica", func(t *testing.T) {
		h, err := repo.HistoricoOrbit(ctx, id, base.Add(-time.Hour), time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if len(h) != 4 {
			t.Fatalf("esperava 4 leituras no histórico, veio %d", len(h))
		}
		for i := 1; i < len(h); i++ {
			if h[i].DetectadoEm.Before(h[i-1].DetectadoEm) {
				t.Fatal("histórico fora de ordem")
			}
		}
		if _, err := repo.HistoricoOrbit(ctx, orbitAleatorio(), base, time.Now()); err != ErrNaoEncontrado {
			t.Fatalf("Orbit inexistente deveria dar ErrNaoEncontrado, veio %v", err)
		}
	})
}

// Princípio VII: a tabela de leituras não tem nenhuma coluna que aponte
// para o usuário que fez a leitura.
func TestLeituraOrbitNaoIdentificaUsuario(t *testing.T) {
	pool := testutil.AbrirPool(t)
	linhas, err := pool.Query(context.Background(), `
		select column_name from information_schema.columns where table_name = 'leitura_orbit'
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer linhas.Close()
	var colunas []string
	for linhas.Next() {
		var c string
		_ = linhas.Scan(&c)
		colunas = append(colunas, c)
	}
	for _, c := range colunas {
		if strings.Contains(c, "usuario") || strings.Contains(c, "email") || strings.Contains(c, "dispositivo") {
			t.Fatalf("leitura_orbit não pode ter a coluna %q (colunas: %v)", c, colunas)
		}
	}
	var fks int
	_ = pool.QueryRow(context.Background(), `
		select count(*) from information_schema.table_constraints tc
		join information_schema.constraint_column_usage ccu on ccu.constraint_name = tc.constraint_name
		where tc.table_name = 'leitura_orbit' and tc.constraint_type = 'FOREIGN KEY' and ccu.table_name = 'usuario'
	`).Scan(&fks)
	if fks != 0 {
		t.Fatal(fmt.Sprintf("leitura_orbit não pode ter chave estrangeira para usuario (%d)", fks))
	}
}

func TestPontoFixo(t *testing.T) {
	pool := testutil.AbrirPool(t)
	ctx := context.Background()
	repo := NovoRepositorio(pool)

	if _, err := repo.CriarPontoFixo(ctx, PontoFixo{Nome: "x", Latitude: -23, Longitude: -46, RaioMetros: 10}); err == nil {
		t.Fatal("raio de 10 m deveria ser recusado")
	}
	p, err := repo.CriarPontoFixo(ctx, PontoFixo{Nome: "Pórtico teste", Latitude: -23.6, Longitude: -46.8, RaioMetros: 300, Simulado: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ObterPontoFixo(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.ApagarPontoFixo(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.ApagarPontoFixo(ctx, p.ID); err != ErrNaoEncontrado {
		t.Fatalf("apagar de novo deveria dar ErrNaoEncontrado, veio %v", err)
	}
	if _, err := repo.ObterPontoFixo(ctx, "nao-e-uuid"); err != ErrNaoEncontrado {
		t.Fatalf("id inválido deveria dar ErrNaoEncontrado, veio %v", err)
	}
}
