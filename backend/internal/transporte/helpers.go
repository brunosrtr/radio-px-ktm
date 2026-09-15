package transporte

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/canal"
)

func responderJSON(w http.ResponseWriter, status int, corpo any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(corpo)
}

func responderErro(w http.ResponseWriter, status int, codigo, mensagem string) {
	responderJSON(w, status, map[string]string{"codigo": codigo, "mensagem": mensagem})
}

// responderErroServico traduz os erros de negócio do pacote canal para os
// códigos de erro do contrato REST (contracts/rest-api.md).
func responderErroServico(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, canal.ErrCanalInvalido):
		responderErro(w, http.StatusUnprocessableEntity, "canal_invalido", err.Error())
	case errors.Is(err, canal.ErrNaoEncontrado):
		responderErro(w, http.StatusNotFound, "nao_encontrado", "canal não encontrado")
	case errors.Is(err, canal.ErrSemPermissao):
		responderErro(w, http.StatusForbidden, "sem_permissao", "ação fora do escopo da sua empresa")
	default:
		responderErro(w, http.StatusInternalServerError, "erro_interno", "erro interno")
	}
}
