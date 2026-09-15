package transporte

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

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

// PosicoesAtuais implementa GET /empresas/{id}/posicoes-atuais (admin) — só
// retorna dados quando {id} é a própria empresa do admin (FR-026).
func (h *PosicoesHandler) PosicoesAtuais(w http.ResponseWriter, r *http.Request) {
	claims, ok := exigirAdmin(w, r)
	if !ok {
		return
	}

	motoristas, err := h.servico.PosicoesAtuaisPorEmpresa(r.Context(), claims.EmpresaID, chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, posicao.ErrSemPermissao) {
			responderErro(w, http.StatusForbidden, "sem_permissao", "acesso restrito à própria empresa")
			return
		}
		responderErro(w, http.StatusInternalServerError, "erro_interno", "não foi possível consultar as posições")
		return
	}

	resultado := make([]map[string]any, 0, len(motoristas))
	for _, m := range motoristas {
		resultado = append(resultado, map[string]any{
			"usuario_id":     m.UsuarioID,
			"nome":           m.Nome,
			"latitude":       m.Latitude,
			"longitude":      m.Longitude,
			"velocidade_kmh": m.VelocidadeKmh,
			"capturado_em":   m.CapturadoEm,
		})
	}
	responderJSON(w, http.StatusOK, map[string]any{"motoristas": resultado})
}

// Trajeto implementa GET /motoristas/{id}/trajeto?de=&ate= (admin) — {id}
// precisa pertencer à mesma empresa do admin autenticado (FR-027). Um
// período sem dados retorna 200 com trajeto vazio, nunca erro.
func (h *PosicoesHandler) Trajeto(w http.ResponseWriter, r *http.Request) {
	claims, ok := exigirAdmin(w, r)
	if !ok {
		return
	}

	de, err := time.Parse(time.RFC3339, r.URL.Query().Get("de"))
	if err != nil {
		responderErro(w, http.StatusBadRequest, "mensagem_invalida", "parâmetro 'de' ausente ou fora do formato RFC3339")
		return
	}
	ate, err := time.Parse(time.RFC3339, r.URL.Query().Get("ate"))
	if err != nil {
		responderErro(w, http.StatusBadRequest, "mensagem_invalida", "parâmetro 'ate' ausente ou fora do formato RFC3339")
		return
	}

	pontos, err := h.servico.TrajetoDoMotorista(r.Context(), claims.EmpresaID, chi.URLParam(r, "id"), de, ate)
	if err != nil {
		if errors.Is(err, posicao.ErrSemPermissao) {
			responderErro(w, http.StatusForbidden, "sem_permissao", "motorista não pertence à sua empresa")
			return
		}
		responderErro(w, http.StatusInternalServerError, "erro_interno", "não foi possível consultar o trajeto")
		return
	}

	trajeto := make([]map[string]any, 0, len(pontos))
	for _, p := range pontos {
		trajeto = append(trajeto, map[string]any{
			"latitude":     p.Latitude,
			"longitude":    p.Longitude,
			"capturado_em": p.CapturadoEm,
		})
	}
	responderJSON(w, http.StatusOK, map[string]any{"trajeto": trajeto})
}
