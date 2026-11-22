package transporte

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/canal"
)

// Detalhar implementa GET /canais/{id}: o canal para quem pode vê-lo. O
// código de convite só vai para quem gerencia o canal.
func (h *CanaisHandler) Detalhar(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsOu401(w, r)
	if !ok {
		return
	}
	d, err := h.servico.Detalhar(r.Context(), chi.URLParam(r, "id"), autorDe(claims))
	if err != nil {
		responderErroServico(w, err)
		return
	}
	corpo := canalParaJSON(&d.CanalDB)
	if !d.PodeGerenciar {
		corpo["codigo_convite"] = nil
	}
	corpo["sou_criador"] = d.SouCriador
	corpo["pode_gerenciar"] = d.PodeGerenciar
	corpo["favorito"] = d.Favorito
	corpo["participantes_atual"] = d.ParticipantesAtual
	responderJSON(w, http.StatusOK, corpo)
}

// Apagar implementa DELETE /canais/{id}.
func (h *CanaisHandler) Apagar(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsOu401(w, r)
	if !ok {
		return
	}
	if err := h.servico.Apagar(r.Context(), chi.URLParam(r, "id"), autorDe(claims)); err != nil {
		responderErroServico(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// NovoConvite implementa POST /canais/{id}/convite.
func (h *CanaisHandler) NovoConvite(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsOu401(w, r)
	if !ok {
		return
	}
	codigo, err := h.servico.NovoConvite(r.Context(), chi.URLParam(r, "id"), autorDe(claims))
	if err != nil {
		responderErroServico(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]string{"codigo_convite": canal.FormatarCodigoConvite(codigo)})
}

// EntrarPorConvite implementa POST /convites/{codigo}: libera o canal
// privado para quem tem o código (ou leu o QR).
func (h *CanaisHandler) EntrarPorConvite(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsOu401(w, r)
	if !ok {
		return
	}
	c, err := h.servico.EntrarPorConvite(r.Context(), chi.URLParam(r, "codigo"), claims.UsuarioID)
	if err != nil {
		if errors.Is(err, canal.ErrNaoEncontrado) {
			responderErro(w, http.StatusNotFound, "convite_invalido", "convite inválido ou expirado")
			return
		}
		responderErroServico(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]any{"id": c.ID, "nome": c.Nome, "limite_participantes": c.LimiteParticipantes})
}

// Participantes implementa GET /canais/{id}/participantes (quem gerencia).
func (h *CanaisHandler) Participantes(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsOu401(w, r)
	if !ok {
		return
	}
	lista, err := h.servico.Participantes(r.Context(), chi.URLParam(r, "id"), autorDe(claims))
	if err != nil {
		responderErroServico(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]any{"participantes": lista})
}

// RemoverParticipante implementa DELETE /canais/{id}/participantes/{usuarioId}.
func (h *CanaisHandler) RemoverParticipante(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsOu401(w, r)
	if !ok {
		return
	}
	if err := h.servico.Remover(r.Context(), chi.URLParam(r, "id"), autorDe(claims), chi.URLParam(r, "usuarioId")); err != nil {
		responderErroServico(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Favoritar implementa PUT /canais/{id}/favorito.
func (h *CanaisHandler) Favoritar(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsOu401(w, r)
	if !ok {
		return
	}
	var corpo struct {
		Favorito *bool `json:"favorito"`
	}
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil || corpo.Favorito == nil {
		responderErro(w, http.StatusBadRequest, "mensagem_invalida", `informe {"favorito": true|false}`)
		return
	}
	if err := h.servico.Favoritar(r.Context(), chi.URLParam(r, "id"), autorDe(claims), *corpo.Favorito); err != nil {
		responderErroServico(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Explorar implementa GET /canais/explorar?q=&lat=&lng=&perto=true (FR-030).
func (h *CanaisHandler) Explorar(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsOu401(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	var lat, lng *float64
	if v, err := strconv.ParseFloat(q.Get("lat"), 64); err == nil {
		lat = &v
	}
	if v, err := strconv.ParseFloat(q.Get("lng"), 64); err == nil {
		lng = &v
	}
	busca := q.Get("q")
	if len(busca) > 60 {
		busca = busca[:60]
	}
	canais, err := h.servico.Explorar(r.Context(), claims.UsuarioID, busca, lat, lng, q.Get("perto") == "true")
	if err != nil {
		responderErroServico(w, err)
		return
	}
	lista := make([]map[string]any, 0, len(canais))
	for _, c := range canais {
		lista = append(lista, map[string]any{
			"id":                   c.ID,
			"nome":                 c.Nome,
			"descricao":            c.Descricao,
			"limite_participantes": c.LimiteParticipantes,
			"participantes_atual":  c.ParticipantesAtual,
			"geocerca_ativa":       c.GeocercaAtiva,
			"raio_metros":          c.RaioMetros,
			"distancia_metros":     c.DistanciaMetros,
			"favorito":             c.Favorito,
		})
	}
	responderJSON(w, http.StatusOK, map[string]any{"canais": lista})
}
