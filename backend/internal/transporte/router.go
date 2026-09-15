// Package transporte reúne o roteador HTTP, os handlers REST e os DTOs
// expostos pela API.
package transporte

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/canal"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/config"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/ws"
)

// NovoRoteador monta o roteador HTTP base do backend. /health fica fora do
// grupo autenticado de propósito, pois é o critério de aceite da Etapa 1
// (fundação) e não deve depender de token.
func NovoRoteador(cfg config.Config, pool *storage.Pool) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", handlerHealth(pool))

	gerenciadorCanais := canal.NovoGerenciador()
	repositorioCanais := canal.NovoRepositorio(pool)
	wsHandler := ws.NovoHandler(gerenciadorCanais, repositorioCanais)

	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(cfg.JWTSecret))
		r.Get("/ws", wsHandler.ServeHTTP)
	})

	return r
}

func handlerHealth(pool *storage.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"codigo":   "banco_indisponivel",
				"mensagem": "não foi possível conectar ao Postgres",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}
