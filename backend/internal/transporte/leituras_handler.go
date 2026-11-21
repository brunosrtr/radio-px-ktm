package transporte

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/orbit"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/usuario"
)

// LeiturasHandler implementa as rotas do ponto de transmissão: envio de
// leituras Orbit, sinal de vida e contadores de contribuição.
type LeiturasHandler struct {
	servico  *orbit.Servico
	usuarios *usuario.Repositorio
}

// NovoLeiturasHandler cria o handler.
func NovoLeiturasHandler(servico *orbit.Servico, usuarios *usuario.Repositorio) *LeiturasHandler {
	return &LeiturasHandler{servico: servico, usuarios: usuarios}
}

// consentiu responde 403 e devolve false quando o usuário não autorizou
// atuar como ponto de transmissão (FR-028).
func (h *LeiturasHandler) consentiu(w http.ResponseWriter, r *http.Request, usuarioID string) bool {
	tem, err := h.usuarios.TemConsentimento(r.Context(), usuarioID, usuario.ConsentimentoPontoTransmissao)
	if err != nil {
		responderErro(w, http.StatusInternalServerError, "erro_interno", "não foi possível verificar o consentimento")
		return false
	}
	if !tem {
		responderErro(w, http.StatusForbidden, "sem_consentimento", "ative a função de ponto de transmissão para enviar leituras")
		return false
	}
	return true
}

// Receber implementa POST /leituras (FR-004, FR-009). O app remove da fila
// local todas as leituras do lote ao receber 200 — inclusive as duplicadas.
func (h *LeiturasHandler) Receber(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsDoContexto(r.Context())
	if !ok {
		responderErro(w, http.StatusUnauthorized, "token_invalido", "token ausente ou inválido")
		return
	}
	if !h.consentiu(w, r, claims.UsuarioID) {
		return
	}
	var corpo struct {
		Leituras []orbit.LeituraRecebida `json:"leituras"`
	}
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
		responderErro(w, http.StatusUnprocessableEntity, "lote_invalido", "corpo da requisição inválido")
		return
	}

	resultado, err := h.servico.IngerirLeituras(r.Context(), claims.UsuarioID, corpo.Leituras)
	switch {
	case errors.Is(err, orbit.ErrLoteInvalido):
		responderErro(w, http.StatusUnprocessableEntity, "lote_invalido", mensagemDoErro(err))
		return
	case err != nil:
		log.Printf("leituras: erro ao ingerir lote: %v", err)
		responderErro(w, http.StatusInternalServerError, "erro_interno", "não foi possível registrar as leituras")
		return
	}
	// FR-011: nenhum dado do Orbit volta para o app.
	responderJSON(w, http.StatusOK, map[string]int{"aceitas": resultado.Aceitas, "duplicadas": resultado.Duplicadas})
}

// Sinal implementa POST /ponto/sinal: o app avisa que a função de ponto de
// transmissão está ativa, mesmo sem leituras (FR-018).
func (h *LeiturasHandler) Sinal(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsDoContexto(r.Context())
	if !ok {
		responderErro(w, http.StatusUnauthorized, "token_invalido", "token ausente ou inválido")
		return
	}
	if !h.consentiu(w, r, claims.UsuarioID) {
		return
	}
	h.servico.RegistrarSinal(claims.UsuarioID)
	w.WriteHeader(http.StatusNoContent)
}

// Contribuicao implementa GET /me/contribuicao (FR-040).
func (h *LeiturasHandler) Contribuicao(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsDoContexto(r.Context())
	if !ok {
		responderErro(w, http.StatusUnauthorized, "token_invalido", "token ausente ou inválido")
		return
	}
	hoje, semana, total, err := h.servico.Contribuicao(r.Context(), claims.UsuarioID)
	if err != nil {
		responderErro(w, http.StatusInternalServerError, "erro_interno", "não foi possível calcular a contribuição")
		return
	}
	ativo, err := h.usuarios.TemConsentimento(r.Context(), claims.UsuarioID, usuario.ConsentimentoPontoTransmissao)
	if err != nil {
		responderErro(w, http.StatusInternalServerError, "erro_interno", "não foi possível verificar o consentimento")
		return
	}
	responderJSON(w, http.StatusOK, map[string]any{"hoje": hoje, "semana": semana, "total": total, "ponto_ativo": ativo})
}
