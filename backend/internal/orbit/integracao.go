package orbit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Integracao é o ponto único e substituível de entrega das leituras à KTM
// (FR-010, Princípio VI). Quando a KTM fornecer sua API, basta uma nova
// implementação — sem mudanças no app nem no restante do backend.
type Integracao interface {
	Encaminhar(ctx context.Context, leituras []LeituraRegistrada, atualizados []OrbitResumo) error
}

// IntegracaoPainel publica as leituras no painel Rede Orbit ao vivo — é a
// integração usada na demonstração.
type IntegracaoPainel struct {
	Broadcaster *Broadcaster
}

// Encaminhar implementa Integracao.
func (i IntegracaoPainel) Encaminhar(_ context.Context, leituras []LeituraRegistrada, atualizados []OrbitResumo) error {
	for _, l := range leituras {
		i.Broadcaster.Publicar(Evento{Tipo: "leitura", Dados: l})
	}
	for _, o := range atualizados {
		i.Broadcaster.Publicar(Evento{Tipo: "orbit_atualizado", Dados: o})
	}
	return nil
}

// IntegracaoHTTP envia as leituras por POST JSON para uma URL da KTM. É um
// esqueleto: o formato definitivo depende da API que a KTM fornecer (spec,
// Pendência 3).
type IntegracaoHTTP struct {
	URL     string
	Cliente *http.Client
}

// NovaIntegracaoHTTP cria a integração com timeout curto.
func NovaIntegracaoHTTP(url string) *IntegracaoHTTP {
	return &IntegracaoHTTP{URL: url, Cliente: &http.Client{Timeout: 10 * time.Second}}
}

// Encaminhar implementa Integracao.
func (i *IntegracaoHTTP) Encaminhar(ctx context.Context, leituras []LeituraRegistrada, _ []OrbitResumo) error {
	corpo, err := json.Marshal(map[string]any{"leituras": leituras})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, i.URL, bytes.NewReader(corpo))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := i.Cliente.Do(req)
	if err != nil {
		return fmt.Errorf("orbit: erro ao encaminhar leituras à KTM: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("orbit: KTM respondeu %d", resp.StatusCode)
	}
	return nil
}
