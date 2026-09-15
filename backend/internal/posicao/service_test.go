package posicao_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/posicao"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage/testutil"
)

func TestIngerirLoteInsereHistoricoEAtualizaPosicaoAtualCondicionalmente(t *testing.T) {
	pool := testutil.AbrirPool(t)
	ctx := context.Background()

	empresaID := testutil.CriarEmpresa(t, pool)
	motoristaID := testutil.CriarUsuario(t, pool, empresaID, "motorista")

	repositorio := posicao.NovoRepositorio(pool)
	servico := posicao.NovoServico(repositorio, nil)

	agora := time.Now().UTC().Truncate(time.Second)

	lote := []posicao.Ponto{
		{Latitude: -28.40, Longitude: -52.10, CapturadoEm: agora},
		{Latitude: -28.41, Longitude: -52.11, CapturadoEm: agora.Add(30 * time.Second)},
	}
	if err := servico.IngerirLote(ctx, motoristaID, lote); err != nil {
		t.Fatalf("erro ao ingerir lote: %v", err)
	}

	if total := contarHistorico(t, pool, motoristaID); total != 2 {
		t.Fatalf("esperava 2 linhas no histórico, veio %d", total)
	}

	atual, err := repositorio.ObterAtual(ctx, motoristaID)
	if err != nil {
		t.Fatalf("erro ao buscar posição atual: %v", err)
	}
	if !atual.CapturadoEm.Equal(lote[1].CapturadoEm) {
		t.Fatalf("esperava posicao_atual com capturado_em %v, veio %v", lote[1].CapturadoEm, atual.CapturadoEm)
	}

	// Lote atrasado (capturado_em anterior ao já gravado): acumula no
	// histórico, mas não pode regredir posicao_atual (research.md §8).
	loteAtrasado := []posicao.Ponto{
		{Latitude: -99, Longitude: -99, CapturadoEm: agora.Add(-time.Hour)},
	}
	if err := servico.IngerirLote(ctx, motoristaID, loteAtrasado); err != nil {
		t.Fatalf("erro ao ingerir lote atrasado: %v", err)
	}

	if total := contarHistorico(t, pool, motoristaID); total != 3 {
		t.Fatalf("esperava 3 linhas no histórico após o lote atrasado, veio %d", total)
	}

	atual, err = repositorio.ObterAtual(ctx, motoristaID)
	if err != nil {
		t.Fatalf("erro ao buscar posição atual após lote atrasado: %v", err)
	}
	if !atual.CapturadoEm.Equal(lote[1].CapturadoEm) {
		t.Fatalf("posicao_atual não deveria ter regredido: esperava %v, veio %v", lote[1].CapturadoEm, atual.CapturadoEm)
	}
}

func TestIngerirLoteRecusaLoteVazio(t *testing.T) {
	pool := testutil.AbrirPool(t)
	servico := posicao.NovoServico(posicao.NovoRepositorio(pool), nil)

	if err := servico.IngerirLote(context.Background(), "usuario-inexistente", nil); !errors.Is(err, posicao.ErrLoteInvalido) {
		t.Fatalf("esperava ErrLoteInvalido para lote vazio, veio: %v", err)
	}
}

func contarHistorico(t *testing.T, pool *storage.Pool, usuarioID string) int {
	t.Helper()
	var total int
	if err := pool.QueryRow(context.Background(), `select count(*) from posicao where usuario_id = $1`, usuarioID).Scan(&total); err != nil {
		t.Fatalf("erro ao contar histórico de posições: %v", err)
	}
	return total
}
