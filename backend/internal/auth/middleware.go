package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type chaveContexto int

const chaveClaims chaveContexto = iota

// Middleware retorna um middleware HTTP que valida o JWT (do header
// Authorization ou do parâmetro de query "token", usado no upgrade de
// WebSocket) e injeta os claims no contexto da requisição. Responde 401 antes
// de qualquer lógica de negócio quando o token está ausente ou é inválido
// (RNF11).
func Middleware(segredo string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extrairToken(r)
			if token == "" {
				responderNaoAutorizado(w)
				return
			}

			claims, err := ValidarToken(segredo, token)
			if err != nil {
				responderNaoAutorizado(w)
				return
			}

			ctx := context.WithValue(r.Context(), chaveClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsDoContexto recupera os claims injetados pelo Middleware.
func ClaimsDoContexto(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(chaveClaims).(*Claims)
	return claims, ok
}

func extrairToken(r *http.Request) string {
	cabecalho := r.Header.Get("Authorization")
	if strings.HasPrefix(cabecalho, "Bearer ") {
		return strings.TrimPrefix(cabecalho, "Bearer ")
	}
	return r.URL.Query().Get("token")
}

func responderNaoAutorizado(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"codigo":   "token_invalido",
		"mensagem": "token ausente, expirado ou malformado",
	})
}
