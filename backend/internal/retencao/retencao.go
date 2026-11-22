// Package retencao apaga dados que já passaram do prazo de guarda: leituras
// Orbit e histórico de posições com mais de 90 dias (FR-025 da 001,
// data-model da 003) e códigos de recuperação de senha vencidos.
package retencao

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
)

// Prazos de guarda.
const (
	PrazoLeituras   = 90 * 24 * time.Hour
	PrazoPosicoes   = 90 * 24 * time.Hour
	PrazoRecuperado = 24 * time.Hour
)

// Resultado conta o que foi apagado numa rodada.
type Resultado struct {
	Leituras, Posicoes, CodigosRecuperacao int64
}

// Limpar apaga tudo o que passou do prazo, tomando agora como referência.
func Limpar(ctx context.Context, pool *storage.Pool, agora time.Time) (Resultado, error) {
	var r Resultado
	comandos := []struct {
		sql     string
		limite  time.Time
		destino *int64
	}{
		{`delete from leitura_orbit where detectado_em < $1`, agora.Add(-PrazoLeituras), &r.Leituras},
		{`delete from posicao where capturado_em < $1`, agora.Add(-PrazoPosicoes), &r.Posicoes},
		{`delete from recuperacao_senha where expira_em < $1`, agora.Add(-PrazoRecuperado), &r.CodigosRecuperacao},
	}
	for _, c := range comandos {
		tag, err := pool.Exec(ctx, c.sql, c.limite)
		if err != nil {
			return r, fmt.Errorf("retencao: %w", err)
		}
		*c.destino = tag.RowsAffected()
	}
	return r, nil
}

// Rodar executa Limpar ao começar e depois a cada [intervalo], até o
// contexto acabar. Falhas só ficam no log: a próxima rodada tenta de novo.
func Rodar(ctx context.Context, pool *storage.Pool, intervalo time.Duration) {
	rodada := func() {
		r, err := Limpar(ctx, pool, time.Now())
		if err != nil {
			log.Printf("retenção: %v", err)
			return
		}
		if r.Leituras+r.Posicoes+r.CodigosRecuperacao > 0 {
			log.Printf("retenção: apagadas %d leituras, %d posições e %d códigos de recuperação vencidos", r.Leituras, r.Posicoes, r.CodigosRecuperacao)
		}
	}
	rodada()
	ticker := time.NewTicker(intervalo)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rodada()
		}
	}
}
