package orbit

import (
	"errors"
	"testing"
	"time"
)

func TestValidarLote(t *testing.T) {
	agora := time.Date(2026, 11, 21, 15, 0, 0, 0, time.UTC)
	valida := func() LeituraRecebida {
		return LeituraRecebida{OrbitID: "ORBIT-0A1B2C", Latitude: -23.55, Longitude: -46.63, DetectadoEm: agora.Add(-time.Minute)}
	}
	ptr := func(v int) *int { return &v }

	casos := []struct {
		nome   string
		lote   func() []LeituraRecebida
		valido bool
	}{
		{"uma leitura válida", func() []LeituraRecebida { return []LeituraRecebida{valida()} }, true},
		{"lote vazio", func() []LeituraRecebida { return nil }, false},
		{"100 leituras", func() []LeituraRecebida { return repetir(valida(), 100) }, true},
		{"101 leituras", func() []LeituraRecebida { return repetir(valida(), 101) }, false},
		{"id minúsculo", func() []LeituraRecebida { l := valida(); l.OrbitID = "ORBIT-0a1b2c"; return []LeituraRecebida{l} }, false},
		{"id sem prefixo", func() []LeituraRecebida { l := valida(); l.OrbitID = "0A1B2C"; return []LeituraRecebida{l} }, false},
		{"latitude fora do globo", func() []LeituraRecebida { l := valida(); l.Latitude = 91; return []LeituraRecebida{l} }, false},
		{"coordenada zerada (GPS sem fix)", func() []LeituraRecebida { l := valida(); l.Latitude, l.Longitude = 0, 0; return []LeituraRecebida{l} }, false},
		{"sem horário", func() []LeituraRecebida { l := valida(); l.DetectadoEm = time.Time{}; return []LeituraRecebida{l} }, false},
		{"1 min no futuro (relógio adiantado)", func() []LeituraRecebida {
			l := valida()
			l.DetectadoEm = agora.Add(time.Minute)
			return []LeituraRecebida{l}
		}, true},
		{"3 min no futuro", func() []LeituraRecebida {
			l := valida()
			l.DetectadoEm = agora.Add(3 * time.Minute)
			return []LeituraRecebida{l}
		}, false},
		{"6 dias atrás (fila offline)", func() []LeituraRecebida {
			l := valida()
			l.DetectadoEm = agora.Add(-6 * 24 * time.Hour)
			return []LeituraRecebida{l}
		}, true},
		{"8 dias atrás", func() []LeituraRecebida {
			l := valida()
			l.DetectadoEm = agora.Add(-8 * 24 * time.Hour)
			return []LeituraRecebida{l}
		}, false},
		{"rssi absurdo", func() []LeituraRecebida { l := valida(); l.RSSI = ptr(-300); return []LeituraRecebida{l} }, false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			err := ValidarLote(c.lote(), agora)
			if c.valido && err != nil {
				t.Fatalf("esperava válido, veio %v", err)
			}
			if !c.valido && !errors.Is(err, ErrLoteInvalido) {
				t.Fatalf("esperava ErrLoteInvalido, veio %v", err)
			}
		})
	}
}

func repetir(l LeituraRecebida, n int) []LeituraRecebida {
	lote := make([]LeituraRecebida, n)
	for i := range lote {
		lote[i] = l
		lote[i].DetectadoEm = l.DetectadoEm.Add(-time.Duration(i) * time.Second)
	}
	return lote
}
