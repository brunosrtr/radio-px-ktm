package transporte

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/usuario"
)

type CadastrosHandler struct{ usuarios *usuario.Repositorio }

func NovoCadastrosHandler(r *usuario.Repositorio) *CadastrosHandler { return &CadastrosHandler{r} }
func erroDoCadastro(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usuario.ErrCadastroInvalido):
		responderErro(w, 422, "cadastro_invalido", err.Error())
	case errors.Is(err, usuario.ErrCadastroDuplicado):
		responderErro(w, 409, "cadastro_duplicado", "CPF ou placa já cadastrado. Confira os dados ou selecione um veículo existente.")
	case errors.Is(err, usuario.ErrVeiculoNaoAutorizado):
		responderErro(w, 422, "veiculo_invalido", err.Error())
	default:
		responderErro(w, 500, "erro_interno", "Não foi possível concluir o cadastro.")
	}
}
func lerCadastro(w http.ResponseWriter, r *http.Request, destino any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destino); err != nil {
		responderErro(w, 400, "mensagem_invalida", "Confira os campos do cadastro.")
		return false
	}
	return true
}
func (h *CadastrosHandler) CriarMotorista(w http.ResponseWriter, r *http.Request) {
	claims, ok := exigirAdmin(w, r)
	if !ok {
		return
	}
	var d usuario.DadosMotorista
	if !lerCadastro(w, r, &d) {
		return
	}
	m, err := h.usuarios.CriarMotorista(r.Context(), claims.EmpresaID, d)
	if err != nil {
		erroDoCadastro(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	responderJSON(w, http.StatusCreated, m)
}
func (h *CadastrosHandler) ListarMotoristas(w http.ResponseWriter, r *http.Request) {
	claims, ok := exigirAdmin(w, r)
	if !ok {
		return
	}
	lista, err := h.usuarios.ListarMotoristas(r.Context(), claims.EmpresaID)
	if err != nil {
		erroDoCadastro(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	responderJSON(w, 200, map[string]any{"motoristas": lista})
}
func (h *CadastrosHandler) CriarVeiculo(w http.ResponseWriter, r *http.Request) {
	claims, ok := exigirAdmin(w, r)
	if !ok {
		return
	}
	var d usuario.DadosVeiculo
	if !lerCadastro(w, r, &d) {
		return
	}
	v, err := h.usuarios.CriarVeiculo(r.Context(), claims.EmpresaID, d)
	if err != nil {
		erroDoCadastro(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	responderJSON(w, http.StatusCreated, v)
}
func (h *CadastrosHandler) ListarVeiculos(w http.ResponseWriter, r *http.Request) {
	claims, ok := exigirAdmin(w, r)
	if !ok {
		return
	}
	lista, err := h.usuarios.ListarVeiculos(r.Context(), claims.EmpresaID)
	if err != nil {
		erroDoCadastro(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	responderJSON(w, 200, map[string]any{"veiculos": lista})
}
