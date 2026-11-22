package transporte

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/email"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/usuario"
)

// AuthHandler implementa as rotas de conta: cadastro, login, recuperação de
// senha e /me (contracts/rest-api.md da 003-rede-orbit).
type AuthHandler struct {
	usuarios  *usuario.Repositorio
	jwtSecret string
	enviador  email.Enviador
}

// NovoAuthHandler cria o AuthHandler. enviador pode ser nil: sem SMTP, a
// recuperação de senha responde 503.
func NovoAuthHandler(usuarios *usuario.Repositorio, jwtSecret string, enviador email.Enviador) *AuthHandler {
	return &AuthHandler{usuarios: usuarios, jwtSecret: jwtSecret, enviador: enviador}
}

// hashDeReferencia é um bcrypt válido de uma senha aleatória, usado só para
// igualar o tempo de resposta do login quando a conta não existe.
var hashDeReferencia = func() string {
	hash, err := auth.HashSenha("referencia-tempo-login-sem-conta")
	if err != nil {
		panic(err)
	}
	return hash
}()

type corpoDispositivo struct {
	Plataforma string  `json:"plataforma"`
	VersaoApp  string  `json:"versao_app"`
	PushToken  *string `json:"push_token"`
}

// Cadastro implementa POST /auth/cadastro (FR-023): cria a conta do público
// e já devolve a sessão, como o login.
func (h *AuthHandler) Cadastro(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		usuario.DadosCadastroPublico
		Dispositivo *corpoDispositivo `json:"dispositivo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
		responderErro(w, http.StatusBadRequest, "mensagem_invalida", "corpo da requisição inválido")
		return
	}

	u, err := h.usuarios.CriarUsuarioPublico(r.Context(), corpo.DadosCadastroPublico)
	switch {
	case errors.Is(err, usuario.ErrEmailEmUso):
		responderErro(w, http.StatusConflict, "email_em_uso", "este e-mail já tem conta")
		return
	case errors.Is(err, usuario.ErrDadosInvalidos):
		responderErro(w, http.StatusUnprocessableEntity, "dados_invalidos", mensagemDoErro(err))
		return
	case err != nil:
		log.Printf("auth: erro no cadastro: %v", err)
		responderErro(w, http.StatusInternalServerError, "erro_interno", "não foi possível criar a conta")
		return
	}

	h.responderSessao(w, r, http.StatusCreated, u, corpo.Dispositivo)
}

// Login autentica por e-mail (contas do público) ou login/CPF (contas de
// empresa) e devolve um JWT. Aceita o campo antigo "login" para não quebrar
// versões do app anteriores à 003.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Identificador string            `json:"identificador"`
		Login         string            `json:"login"`
		Senha         string            `json:"senha"`
		Dispositivo   *corpoDispositivo `json:"dispositivo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
		responderErro(w, http.StatusBadRequest, "mensagem_invalida", "corpo da requisição inválido")
		return
	}
	identificador := corpo.Identificador
	if identificador == "" {
		identificador = corpo.Login
	}

	u, err := h.usuarios.BuscarPorIdentificador(r.Context(), identificador)
	if err != nil {
		// Mesmo custo de bcrypt de uma conta existente, para que o tempo de
		// resposta não revele quais e-mails/logins estão cadastrados.
		auth.VerificarSenha(hashDeReferencia, corpo.Senha)
		responderErro(w, http.StatusUnauthorized, "credenciais_invalidas", "e-mail, login ou senha incorretos")
		return
	}
	if !auth.VerificarSenha(u.SenhaHash, corpo.Senha) {
		responderErro(w, http.StatusUnauthorized, "credenciais_invalidas", "e-mail, login ou senha incorretos")
		return
	}

	h.responderSessao(w, r, http.StatusOK, u, corpo.Dispositivo)
}

func (h *AuthHandler) responderSessao(w http.ResponseWriter, r *http.Request, status int, u *usuario.Usuario, dispositivo *corpoDispositivo) {
	token, err := auth.GerarToken(h.jwtSecret, u.ID, u.EmpresaID, auth.Papel(u.Papel))
	if err != nil {
		responderErro(w, http.StatusInternalServerError, "erro_interno", "não foi possível gerar o token")
		return
	}

	if dispositivo != nil && dispositivo.Plataforma != "" {
		if err := h.usuarios.RegistrarDispositivo(
			r.Context(), u.ID, dispositivo.Plataforma, dispositivo.VersaoApp, dispositivo.PushToken,
		); err != nil {
			// Não crítico para a sessão em si — só registra e segue.
			log.Printf("auth: erro ao registrar dispositivo: %v", err)
		}
	}

	responderJSON(w, status, map[string]any{
		"token": token,
		"usuario": map[string]any{
			"id":          u.ID,
			"nome":        u.Nome,
			"apelido":     u.NomeNoRadio(),
			"papel":       u.Papel,
			"tem_empresa": u.TemEmpresa(),
		},
	})
}

// RecuperarSenha implementa POST /auth/recuperar-senha (FR-025). Responde
// 202 sempre — exista ou não a conta — e envia o e-mail em segundo plano,
// para que nem a resposta nem o tempo dela revelem e-mails cadastrados.
func (h *AuthHandler) RecuperarSenha(w http.ResponseWriter, r *http.Request) {
	if h.enviador == nil {
		responderErro(w, http.StatusServiceUnavailable, "recuperacao_indisponivel", "a recuperação de senha não está configurada neste servidor")
		return
	}
	var corpo struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
		responderErro(w, http.StatusBadRequest, "mensagem_invalida", "corpo da requisição inválido")
		return
	}

	go func(endereco string) {
		ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancelar()
		if err := h.usuarios.SolicitarRecuperacao(ctx, endereco, h.enviador); err != nil {
			log.Printf("auth: erro ao solicitar recuperação de senha: %v", err)
		}
	}(corpo.Email)

	responderJSON(w, http.StatusAccepted, map[string]string{
		"mensagem": "se o e-mail tiver conta, enviamos um código",
	})
}

// RedefinirSenha implementa POST /auth/redefinir-senha.
func (h *AuthHandler) RedefinirSenha(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Email     string `json:"email"`
		Codigo    string `json:"codigo"`
		NovaSenha string `json:"nova_senha"`
	}
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
		responderErro(w, http.StatusBadRequest, "mensagem_invalida", "corpo da requisição inválido")
		return
	}

	err := h.usuarios.RedefinirSenha(r.Context(), corpo.Email, strings.TrimSpace(corpo.Codigo), corpo.NovaSenha)
	switch {
	case errors.Is(err, usuario.ErrDadosInvalidos):
		responderErro(w, http.StatusUnprocessableEntity, "dados_invalidos", mensagemDoErro(err))
	case errors.Is(err, usuario.ErrCodigoInvalido):
		responderErro(w, http.StatusBadRequest, "codigo_invalido", "código inválido ou expirado; peça um novo")
	case err != nil:
		log.Printf("auth: erro ao redefinir senha: %v", err)
		responderErro(w, http.StatusInternalServerError, "erro_interno", "não foi possível redefinir a senha")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// Me retorna os dados do usuário autenticado e seus consentimentos vigentes.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsDoContexto(r.Context())
	if !ok {
		responderErro(w, http.StatusUnauthorized, "token_invalido", "token ausente ou inválido")
		return
	}
	h.responderMe(w, r, claims.UsuarioID)
}

func (h *AuthHandler) responderMe(w http.ResponseWriter, r *http.Request, usuarioID string) {
	u, err := h.usuarios.ObterPorID(r.Context(), usuarioID)
	if err != nil {
		responderErro(w, http.StatusNotFound, "nao_encontrado", "usuário não encontrado")
		return
	}
	consentimentos, err := h.usuarios.ConsentimentosVigentes(r.Context(), u.ID)
	if err != nil {
		responderErro(w, http.StatusInternalServerError, "erro_interno", "não foi possível carregar a conta")
		return
	}

	var empresaID any
	if u.TemEmpresa() {
		empresaID = u.EmpresaID
	}
	var emailUsuario any
	if u.Email != "" {
		emailUsuario = u.Email
	}
	responderJSON(w, http.StatusOK, map[string]any{
		"id":             u.ID,
		"nome":           u.Nome,
		"apelido":        u.NomeNoRadio(),
		"email":          emailUsuario,
		"papel":          u.Papel,
		"empresa_id":     empresaID,
		"tem_empresa":    u.TemEmpresa(),
		"consentimentos": consentimentos,
	})
}

// AtualizarMe implementa PATCH /me (nome e apelido).
func (h *AuthHandler) AtualizarMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsDoContexto(r.Context())
	if !ok {
		responderErro(w, http.StatusUnauthorized, "token_invalido", "token ausente ou inválido")
		return
	}
	var corpo struct {
		Nome    string `json:"nome"`
		Apelido string `json:"apelido"`
	}
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
		responderErro(w, http.StatusBadRequest, "mensagem_invalida", "corpo da requisição inválido")
		return
	}
	_, err := h.usuarios.AtualizarPerfil(r.Context(), claims.UsuarioID, corpo.Nome, corpo.Apelido)
	switch {
	case errors.Is(err, usuario.ErrDadosInvalidos):
		responderErro(w, http.StatusUnprocessableEntity, "dados_invalidos", mensagemDoErro(err))
		return
	case errors.Is(err, usuario.ErrNaoEncontrado):
		responderErro(w, http.StatusNotFound, "nao_encontrado", "usuário não encontrado")
		return
	case err != nil:
		responderErro(w, http.StatusInternalServerError, "erro_interno", "não foi possível atualizar a conta")
		return
	}
	h.responderMe(w, r, claims.UsuarioID)
}

// ExcluirMe implementa DELETE /me (FR-026).
func (h *AuthHandler) ExcluirMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsDoContexto(r.Context())
	if !ok {
		responderErro(w, http.StatusUnauthorized, "token_invalido", "token ausente ou inválido")
		return
	}
	err := h.usuarios.ExcluirConta(r.Context(), claims.UsuarioID)
	switch {
	case errors.Is(err, usuario.ErrNaoEncontrado):
		responderErro(w, http.StatusNotFound, "nao_encontrado", "usuário não encontrado")
	case err != nil:
		log.Printf("auth: erro ao excluir conta: %v", err)
		responderErro(w, http.StatusInternalServerError, "erro_interno", "não foi possível excluir a conta")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// DefinirConsentimento implementa PUT /me/consentimentos/{tipo} (FR-028).
func (h *AuthHandler) DefinirConsentimento(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsDoContexto(r.Context())
	if !ok {
		responderErro(w, http.StatusUnauthorized, "token_invalido", "token ausente ou inválido")
		return
	}
	var corpo struct {
		Aceito *bool `json:"aceito"`
	}
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil || corpo.Aceito == nil {
		responderErro(w, http.StatusBadRequest, "mensagem_invalida", `informe {"aceito": true|false}`)
		return
	}
	err := h.usuarios.DefinirConsentimento(r.Context(), claims.UsuarioID, chi.URLParam(r, "tipo"), *corpo.Aceito)
	switch {
	case errors.Is(err, usuario.ErrConsentimentoInvalido):
		responderErro(w, http.StatusUnprocessableEntity, "consentimento_invalido", "tipo de consentimento inválido ou que não pode ser revogado")
	case err != nil:
		responderErro(w, http.StatusInternalServerError, "erro_interno", "não foi possível registrar o consentimento")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// mensagemDoErro tira o prefixo "usuario: dados inválidos: " das mensagens
// de validação, que são exibidas ao usuário pelo app.
func mensagemDoErro(err error) string {
	mensagem := err.Error()
	if i := strings.LastIndex(mensagem, ": "); i >= 0 {
		return mensagem[i+2:]
	}
	return mensagem
}
