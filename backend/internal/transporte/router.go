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
	"github.com/brunosrtr/radio-px-ktm/backend/internal/geofence"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/notificacao"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/posicao"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/usuario"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/ws"
	"github.com/brunosrtr/radio-px-ktm/backend/painel"
)

// NovoRoteador monta o roteador HTTP base do backend. /health e
// /auth/login ficam fora do grupo autenticado de propósito: o primeiro é o
// critério de aceite da Etapa 1 (fundação), o segundo é como se obtém o
// token em primeiro lugar.
func NovoRoteador(cfg config.Config, pool *storage.Pool) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	r.Get("/health", handlerHealth(pool))

	gerenciadorCanais := canal.NovoGerenciador()
	repositorioCanais := canal.NovoRepositorio(pool)
	repositorioPosicoes := posicao.NovoRepositorio(pool)
	servicoCanais := canal.NovoServico(repositorioCanais, gerenciadorCanais, repositorioPosicoes)
	repositorioUsuarios := usuario.NovoRepositorio(pool)

	adaptadorGeocerca := canal.NovoAdaptadorGeocerca(gerenciadorCanais, repositorioCanais, repositorioUsuarios, notificacao.NotificadorLog{})
	verificadorGeocerca := geofence.NovoVerificador(adaptadorGeocerca)
	broadcasterPosicoes := posicao.NovoBroadcaster()
	servicoPosicoes := posicao.NovoServico(repositorioPosicoes, verificadorGeocerca, broadcasterPosicoes)

	authHandler := NovoAuthHandler(repositorioUsuarios, cfg.JWTSecret)
	cadastrosHandler := NovoCadastrosHandler(repositorioUsuarios)
	canaisHandler := NovoCanaisHandler(servicoCanais)
	posicoesHandler := NovoPosicoesHandler(servicoPosicoes)
	wsHandler := ws.NovoHandler(gerenciadorCanais, servicoCanais, repositorioCanais, repositorioUsuarios)
	painelWsHandler := ws.NovoPainelHandler(broadcasterPosicoes)
	centralHandler := ws.NovaCentralHandler(gerenciadorCanais, repositorioCanais)

	r.Post("/auth/login", authHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(cfg.JWTSecret))

		r.Get("/me", authHandler.Me)
		r.Get("/admin/motoristas", cadastrosHandler.ListarMotoristas)
		r.Post("/admin/motoristas", cadastrosHandler.CriarMotorista)
		r.Get("/admin/veiculos", cadastrosHandler.ListarVeiculos)
		r.Post("/admin/veiculos", cadastrosHandler.CriarVeiculo)

		r.Get("/canais", canaisHandler.Listar)
		r.Post("/canais", canaisHandler.Criar)
		r.Patch("/canais/{id}", canaisHandler.Editar)
		r.Post("/canais/{id}/empresas", canaisHandler.LiberarEmpresa)
		r.Delete("/canais/{id}/empresas/{empresaId}", canaisHandler.RevogarEmpresa)
		r.Put("/canais/{id}/preferencia", canaisHandler.DefinirPreferencia)

		r.Post("/posicoes", posicoesHandler.Receber)
		r.Get("/empresas/{id}/posicoes-atuais", posicoesHandler.PosicoesAtuais)
		r.Get("/motoristas/{id}/trajeto", posicoesHandler.Trajeto)

		r.Get("/ws", wsHandler.ServeHTTP)
		r.Get("/ws/painel", painelWsHandler.ServeHTTP)
		r.Get("/central/canais", centralHandler.Listar)
		r.Get("/ws/central", centralHandler.ServeHTTP)
	})

	r.Handle("/painel/*", http.StripPrefix("/painel/", http.FileServerFS(painel.Arquivos)))

	return r
}

// corsMiddleware libera chamadas de outras origens (painel/app servidos de
// outra porta em desenvolvimento). Seguro com Access-Control-Allow-Origin: *
// porque a API autentica por Bearer token no header, nunca por cookie.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
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
