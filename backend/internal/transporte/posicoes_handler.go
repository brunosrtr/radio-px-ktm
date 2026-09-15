package transporte

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/posicao"
)

// PosicoesHandler implementa POST /posicoes (ingestão em lote).
type PosicoesHandler struct {
	servico *posicao.Servico
}

// NovoPosicoesHandler cria o PosicoesHandler sobre o Servico de posições.
func NovoPosicoesHandler(servico *posicao.Servico) *PosicoesHandler {
	return &PosicoesHandler{servico: servico}
}

type corpoPonto struct {
	Latitude       float64    `json:"latitude"`
	Longitude      float64    `json:"longitude"`
	PrecisaoMetros *float64   `json:"precisao_metros"`
	VelocidadeKmh  *float64   `json:"velocidade_kmh"`
	CapturadoEm    *time.Time `json:"capturado_em"`
}

// Receber implementa POST /posicoes (motorista) — insere o lote em posicao e
// atualiza posicao_atual condicionalmente (contracts/rest-api.md).
func (h *PosicoesHandler) Receber(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsDoContexto(r.Context())
	if !ok {
		responderErro(w, http.StatusUnauthorized, "token_invalido", "token ausente ou inválido")
		return
	}

	var corpo struct {
		Pontos []corpoPonto `json:"pontos"`
	}
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
		responderErro(w, http.StatusBadRequest, "lote_invalido", "corpo da requisição inválido")
		return
	}

	pontos := make([]posicao.Ponto, len(corpo.Pontos))
	for i, p := range corpo.Pontos {
		if p.CapturadoEm == nil {
			responderErro(w, http.StatusBadRequest, "lote_invalido", fmt.Sprintf("pontos[%d].capturado_em é obrigatório", i))
			return
		}
		pontos[i] = posicao.Ponto{
			Latitude:       p.Latitude,
			Longitude:      p.Longitude,
			PrecisaoMetros: p.PrecisaoMetros,
			VelocidadeKmh:  p.VelocidadeKmh,
			CapturadoEm:    *p.CapturadoEm,
		}
	}

	if err := h.servico.IngerirLote(r.Context(), claims.UsuarioID, pontos); err != nil {
		if errors.Is(err, posicao.ErrLoteInvalido) {
			responderErro(w, http.StatusBadRequest, "lote_invalido", err.Error())
			return
		}
		responderErro(w, http.StatusInternalServerError, "erro_interno", "não foi possível registrar as posições")
		return
	}

	responderJSON(w, http.StatusAccepted, map[string]any{"pontos_recebidos": len(pontos)})
}
