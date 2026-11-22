package transporte

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/canal"
)

// CanaisHandler implementa o CRUD de canais, a liberação/revogação de
// empresas parceiras, a preferência de silenciamento e a listagem para
// motoristas (contracts/rest-api.md).
type CanaisHandler struct {
	servico *canal.Servico
}

// NovoCanaisHandler cria o CanaisHandler sobre o Servico de canais.
func NovoCanaisHandler(servico *canal.Servico) *CanaisHandler {
	return &CanaisHandler{servico: servico}
}

type corpoCanal struct {
	Visibilidade        string   `json:"visibilidade"`
	Nome                string   `json:"nome"`
	Descricao           string   `json:"descricao"`
	TipoAcesso          string   `json:"tipo_acesso"`
	LimiteParticipantes int      `json:"limite_participantes"`
	GeocercaAtiva       bool     `json:"geocerca_ativa"`
	CentroLatitude      *float64 `json:"centro_latitude"`
	CentroLongitude     *float64 `json:"centro_longitude"`
	RaioMetros          *int     `json:"raio_metros"`
	// Geocerca no formato do contrato da 003 (o painel usa os campos soltos
	// acima).
	Geocerca *struct {
		Latitude   float64 `json:"latitude"`
		Longitude  float64 `json:"longitude"`
		RaioMetros int     `json:"raio_metros"`
	} `json:"geocerca"`
}

func (c corpoCanal) paraDTO() canal.DTOCanal {
	tipoAcesso := c.TipoAcesso
	if tipoAcesso == "" {
		tipoAcesso = "privado"
	}
	if c.Geocerca != nil {
		lat, lng, raio := c.Geocerca.Latitude, c.Geocerca.Longitude, c.Geocerca.RaioMetros
		c.GeocercaAtiva, c.CentroLatitude, c.CentroLongitude, c.RaioMetros = true, &lat, &lng, &raio
	}
	return canal.DTOCanal{
		Visibilidade:        c.Visibilidade,
		Nome:                c.Nome,
		Descricao:           c.Descricao,
		TipoAcesso:          tipoAcesso,
		LimiteParticipantes: c.LimiteParticipantes,
		GeocercaAtiva:       c.GeocercaAtiva,
		CentroLatitude:      c.CentroLatitude,
		CentroLongitude:     c.CentroLongitude,
		RaioMetros:          c.RaioMetros,
	}
}

// canalParaJSON serializa um canal para quem pode gerenciá-lo — inclui o
// código de convite.
func canalParaJSON(c *canal.CanalDB) map[string]any {
	var empresaID, codigo any
	if c.EmpresaID != "" {
		empresaID = c.EmpresaID
	}
	if c.CodigoConvite != "" {
		codigo = canal.FormatarCodigoConvite(c.CodigoConvite)
	}
	return map[string]any{
		"id":                   c.ID,
		"empresa_id":           empresaID,
		"comunidade":           c.Comunidade(),
		"visibilidade":         c.Visibilidade,
		"codigo_convite":       codigo,
		"nome":                 c.Nome,
		"descricao":            c.Descricao,
		"tipo_acesso":          c.TipoAcesso,
		"limite_participantes": c.LimiteParticipantes,
		"geocerca_ativa":       c.GeocercaAtiva,
		"centro_latitude":      c.CentroLatitude,
		"centro_longitude":     c.CentroLongitude,
		"raio_metros":          c.RaioMetros,
	}
}

func exigirAdmin(w http.ResponseWriter, r *http.Request) (*auth.Claims, bool) {
	claims, ok := auth.ClaimsDoContexto(r.Context())
	if !ok {
		responderErro(w, http.StatusUnauthorized, "token_invalido", "token ausente ou inválido")
		return nil, false
	}
	if claims.Papel != auth.PapelAdmin {
		responderErro(w, http.StatusForbidden, "sem_permissao", "ação restrita a administradores")
		return nil, false
	}
	return claims, true
}

func autorDe(claims *auth.Claims) canal.Autor {
	return canal.Autor{
		UsuarioID:    claims.UsuarioID,
		EmpresaID:    claims.EmpresaID,
		AdminEmpresa: claims.Papel == auth.PapelAdmin && claims.TemEmpresa(),
	}
}

func claimsOu401(w http.ResponseWriter, r *http.Request) (*auth.Claims, bool) {
	claims, ok := auth.ClaimsDoContexto(r.Context())
	if !ok {
		responderErro(w, http.StatusUnauthorized, "token_invalido", "token ausente ou inválido")
	}
	return claims, ok
}

// Criar implementa POST /canais: admin de empresa cria canal da empresa
// (001); qualquer outra conta cria canal da comunidade (FR-029).
func (h *CanaisHandler) Criar(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsOu401(w, r)
	if !ok {
		return
	}

	var corpo corpoCanal
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
		responderErro(w, http.StatusBadRequest, "mensagem_invalida", "corpo da requisição inválido")
		return
	}

	c, err := h.servico.CriarComo(r.Context(), autorDe(claims), corpo.paraDTO())
	if err != nil {
		responderErroServico(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, canalParaJSON(c))
}

// Editar implementa PATCH /canais/{id}: criador do canal da comunidade ou
// admin da empresa dona (FR-031).
func (h *CanaisHandler) Editar(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsOu401(w, r)
	if !ok {
		return
	}

	var corpo corpoCanal
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
		responderErro(w, http.StatusBadRequest, "mensagem_invalida", "corpo da requisição inválido")
		return
	}

	c, err := h.servico.EditarComo(r.Context(), chi.URLParam(r, "id"), autorDe(claims), corpo.paraDTO())
	if err != nil {
		responderErroServico(w, err)
		return
	}
	responderJSON(w, http.StatusOK, canalParaJSON(c))
}

// LiberarEmpresa implementa POST /canais/{id}/empresas (admin).
func (h *CanaisHandler) LiberarEmpresa(w http.ResponseWriter, r *http.Request) {
	claims, ok := exigirAdmin(w, r)
	if !ok {
		return
	}

	var corpo struct {
		EmpresaID string `json:"empresa_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil || corpo.EmpresaID == "" {
		responderErro(w, http.StatusBadRequest, "mensagem_invalida", "empresa_id é obrigatório")
		return
	}

	if err := h.servico.LiberarEmpresa(r.Context(), chi.URLParam(r, "id"), claims.EmpresaID, corpo.EmpresaID); err != nil {
		responderErroServico(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RevogarEmpresa implementa DELETE /canais/{id}/empresas/{empresaId} (admin).
func (h *CanaisHandler) RevogarEmpresa(w http.ResponseWriter, r *http.Request) {
	claims, ok := exigirAdmin(w, r)
	if !ok {
		return
	}

	err := h.servico.RevogarEmpresa(r.Context(), chi.URLParam(r, "id"), claims.EmpresaID, chi.URLParam(r, "empresaId"))
	if err != nil {
		responderErroServico(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DefinirPreferencia implementa PUT /canais/{id}/preferencia (motorista).
func (h *CanaisHandler) DefinirPreferencia(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsDoContexto(r.Context())
	if !ok {
		responderErro(w, http.StatusUnauthorized, "token_invalido", "token ausente ou inválido")
		return
	}

	var corpo struct {
		Silenciado bool `json:"silenciado"`
	}
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
		responderErro(w, http.StatusBadRequest, "mensagem_invalida", "corpo da requisição inválido")
		return
	}

	canalID := chi.URLParam(r, "id")
	if err := h.servico.DefinirPreferencia(r.Context(), claims.UsuarioID, canalID, corpo.Silenciado); err != nil {
		responderErroServico(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]any{"silenciado": corpo.Silenciado})
}

// Listar implementa GET /canais (motorista) — canais autorizados para a
// empresa do motorista (RNF12).
func (h *CanaisHandler) Listar(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsDoContexto(r.Context())
	if !ok {
		responderErro(w, http.StatusUnauthorized, "token_invalido", "token ausente ou inválido")
		return
	}

	listados, err := h.servico.ListarParaMotorista(r.Context(), claims.EmpresaID, claims.UsuarioID)
	if err != nil {
		responderErro(w, http.StatusInternalServerError, "erro_interno", "não foi possível listar os canais")
		return
	}

	canais := make([]map[string]any, 0, len(listados))
	for _, c := range listados {
		canais = append(canais, map[string]any{
			"id":                   c.ID,
			"nome":                 c.Nome,
			"tipo_acesso":          c.TipoAcesso,
			"limite_participantes": c.LimiteParticipantes,
			"participantes_atual":  c.ParticipantesAtual,
			"geocerca_ativa":       c.GeocercaAtiva,
			"silenciado":           c.Silenciado,
			"descricao":            c.Descricao,
			"comunidade":           c.Comunidade(),
			"visibilidade":         c.Visibilidade,
			"favorito":             c.Favorito,
			"sou_criador":          c.SouCriador,
		})
	}
	responderJSON(w, http.StatusOK, map[string]any{"canais": canais})
}
