package transporte

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/orbit"
)

// RedeHandler implementa as rotas do painel Rede Orbit, restritas ao papel
// admin_ktm (FR-017 a FR-022).
type RedeHandler struct {
	servico *orbit.Servico
}

// NovoRedeHandler cria o handler.
func NovoRedeHandler(servico *orbit.Servico) *RedeHandler {
	return &RedeHandler{servico: servico}
}

// SomenteAdminKTM recusa com 403 quem não é administrador da KTM.
func SomenteAdminKTM(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsDoContexto(r.Context())
		if !ok {
			responderErro(w, http.StatusUnauthorized, "token_invalido", "token ausente ou inválido")
			return
		}
		if claims.Papel != auth.PapelAdminKTM {
			responderErro(w, http.StatusForbidden, "sem_permissao", "o painel Rede Orbit é restrito à KTM")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func responderErroRede(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, orbit.ErrNaoEncontrado):
		responderErro(w, http.StatusNotFound, "nao_encontrado", "não encontrado")
	case errors.Is(err, orbit.ErrDadosInvalidos), errors.Is(err, orbit.ErrLoteInvalido):
		responderErro(w, http.StatusUnprocessableEntity, "dados_invalidos", mensagemDoErro(err))
	default:
		responderErro(w, http.StatusInternalServerError, "erro_interno", "erro interno")
	}
}

// Orbits implementa GET /rede/orbits.
func (h *RedeHandler) Orbits(w http.ResponseWriter, r *http.Request) {
	orbits, err := h.servico.Repositorio().Orbits(r.Context())
	if err != nil {
		responderErroRede(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]any{"orbits": orbits})
}

// Leituras implementa GET /rede/orbits/{id}/leituras?de=&ate= (padrão:
// últimas 24 h).
func (h *RedeHandler) Leituras(w http.ResponseWriter, r *http.Request) {
	ate := time.Now()
	de := ate.Add(-24 * time.Hour)
	for nome, destino := range map[string]*time.Time{"de": &de, "ate": &ate} {
		if valor := r.URL.Query().Get(nome); valor != "" {
			t, err := time.Parse(time.RFC3339, valor)
			if err != nil {
				responderErro(w, http.StatusBadRequest, "periodo_invalido", "use datas no formato RFC 3339")
				return
			}
			*destino = t
		}
	}
	historico, err := h.servico.Repositorio().HistoricoOrbit(r.Context(), chi.URLParam(r, "id"), de, ate)
	if err != nil {
		responderErroRede(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]any{"leituras": historico})
}

// PontosFixos implementa GET /rede/pontos-fixos.
func (h *RedeHandler) PontosFixos(w http.ResponseWriter, r *http.Request) {
	pontos, err := h.servico.Repositorio().PontosFixos(r.Context())
	if err != nil {
		responderErroRede(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]any{"pontos_fixos": pontos})
}

// CriarPontoFixo implementa POST /rede/pontos-fixos.
func (h *RedeHandler) CriarPontoFixo(w http.ResponseWriter, r *http.Request) {
	ponto := orbit.PontoFixo{Simulado: true}
	if err := json.NewDecoder(r.Body).Decode(&ponto); err != nil {
		responderErro(w, http.StatusBadRequest, "mensagem_invalida", "corpo da requisição inválido")
		return
	}
	criado, err := h.servico.Repositorio().CriarPontoFixo(r.Context(), ponto)
	if err != nil {
		responderErroRede(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, criado)
}

// ApagarPontoFixo implementa DELETE /rede/pontos-fixos/{id}.
func (h *RedeHandler) ApagarPontoFixo(w http.ResponseWriter, r *http.Request) {
	if err := h.servico.Repositorio().ApagarPontoFixo(r.Context(), chi.URLParam(r, "id")); err != nil {
		responderErroRede(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Simular implementa POST /rede/pontos-fixos/{id}/simular: gera uma leitura
// de origem "fixo" para mostrar na demonstração como é a cobertura atual.
func (h *RedeHandler) Simular(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		OrbitID string `json:"orbit_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
		responderErro(w, http.StatusBadRequest, "mensagem_invalida", "corpo da requisição inválido")
		return
	}
	resultado, err := h.servico.SimularPassagem(r.Context(), chi.URLParam(r, "id"), corpo.OrbitID)
	if err != nil {
		responderErroRede(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]int{"aceitas": resultado.Aceitas})
}
