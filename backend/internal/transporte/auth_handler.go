package transporte

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/usuario"
)

// AuthHandler implementa POST /auth/login e GET /me.
type AuthHandler struct {
	usuarios  *usuario.Repositorio
	jwtSecret string
}

// NovoAuthHandler cria o AuthHandler sobre o repositório de usuários e o
// segredo usado para assinar tokens.
func NovoAuthHandler(usuarios *usuario.Repositorio, jwtSecret string) *AuthHandler {
	return &AuthHandler{usuarios: usuarios, jwtSecret: jwtSecret}
}

// Login autentica por login/senha e devolve um JWT (FR-020, RF21).
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Login       string `json:"login"`
		Senha       string `json:"senha"`
		Dispositivo *struct {
			Plataforma string  `json:"plataforma"`
			VersaoApp  string  `json:"versao_app"`
			PushToken  *string `json:"push_token"`
		} `json:"dispositivo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
		responderErro(w, http.StatusBadRequest, "mensagem_invalida", "corpo da requisição inválido")
		return
	}

	u, err := h.usuarios.BuscarPorLogin(r.Context(), usuario.NormalizarLogin(corpo.Login))
	if err != nil {
		responderErro(w, http.StatusUnauthorized, "credenciais_invalidas", "login ou senha incorretos")
		return
	}
	if !auth.VerificarSenha(u.SenhaHash, corpo.Senha) {
		responderErro(w, http.StatusUnauthorized, "credenciais_invalidas", "login ou senha incorretos")
		return
	}

	token, err := auth.GerarToken(h.jwtSecret, u.ID, u.EmpresaID, auth.Papel(u.Papel))
	if err != nil {
		responderErro(w, http.StatusInternalServerError, "erro_interno", "não foi possível gerar o token")
		return
	}

	if corpo.Dispositivo != nil && corpo.Dispositivo.Plataforma != "" {
		if err := h.usuarios.RegistrarDispositivo(
			r.Context(), u.ID, corpo.Dispositivo.Plataforma, corpo.Dispositivo.VersaoApp, corpo.Dispositivo.PushToken,
		); err != nil {
			// Não crítico para o login em si — só registra e segue.
			log.Printf("auth: erro ao registrar dispositivo: %v", err)
		}
	}

	responderJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"usuario": map[string]any{
			"id":    u.ID,
			"nome":  u.Nome,
			"papel": u.Papel,
		},
	})
}

// Me retorna os dados do usuário autenticado.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsDoContexto(r.Context())
	if !ok {
		responderErro(w, http.StatusUnauthorized, "token_invalido", "token ausente ou inválido")
		return
	}

	u, err := h.usuarios.ObterPorID(r.Context(), claims.UsuarioID)
	if err != nil {
		responderErro(w, http.StatusNotFound, "nao_encontrado", "usuário não encontrado")
		return
	}

	responderJSON(w, http.StatusOK, map[string]any{
		"id":         u.ID,
		"nome":       u.Nome,
		"papel":      u.Papel,
		"empresa_id": u.EmpresaID,
	})
}
