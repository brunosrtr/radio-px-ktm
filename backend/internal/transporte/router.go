// Package transporte reúne o roteador HTTP, os handlers REST e os DTOs
// expostos pela API.
package transporte

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/canal"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/config"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/email"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/geofence"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/notificacao"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/orbit"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/posicao"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/usuario"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/ws"
	"github.com/brunosrtr/radio-px-ktm/backend/painel"
)

// Dependencias são os colaboradores externos do roteador que os testes
// substituem.
type Dependencias struct {
	// Enviador de e-mail da recuperação de senha; nil desativa a recuperação.
	Enviador email.Enviador
	// LimitadorContas protege cadastro e recuperação de senha; nil usa o
	// padrão de 5 tentativas por IP a cada 10 minutos.
	LimitadorContas *Limitador
}

// NovoRoteador monta o roteador HTTP do backend com as dependências
// derivadas da configuração (SMTP real ou nenhum).
func NovoRoteador(cfg config.Config, pool *storage.Pool) chi.Router {
	return NovoRoteadorCom(cfg, pool, Dependencias{Enviador: email.NovoEnviador(cfg)})
}

// NovoRoteadorCom monta o roteador HTTP base do backend. /health e as rotas
// de conta (cadastro, login, recuperação de senha) ficam fora do grupo
// autenticado de propósito: são como se obtém o token em primeiro lugar.
func NovoRoteadorCom(cfg config.Config, pool *storage.Pool, deps Dependencias) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	// O endereço principal abre o painel; as rotas da API continuam explícitas.
	abrirPainel := func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/painel/", http.StatusFound)
	}
	r.Get("/", abrirPainel)
	r.Get("/painel", abrirPainel)

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

	authHandler := NovoAuthHandler(repositorioUsuarios, cfg.JWTSecret, deps.Enviador)
	cadastrosHandler := NovoCadastrosHandler(repositorioUsuarios)
	canaisHandler := NovoCanaisHandler(servicoCanais)
	posicoesHandler := NovoPosicoesHandler(servicoPosicoes)
	wsHandler := ws.NovoHandler(gerenciadorCanais, servicoCanais, repositorioCanais, repositorioUsuarios)
	painelWsHandler := ws.NovoPainelHandler(broadcasterPosicoes)
	centralHandler := ws.NovaCentralHandler(gerenciadorCanais, repositorioCanais)

	// Rede Orbit (003-rede-orbit): o painel ao vivo é sempre uma integração;
	// a da KTM entra quando KTM_INTEGRACAO_URL estiver configurada.
	broadcasterRede := orbit.NovoBroadcaster()
	integracoes := []orbit.Integracao{orbit.IntegracaoPainel{Broadcaster: broadcasterRede}}
	if cfg.KTMIntegracaoURL != "" {
		integracoes = append(integracoes, orbit.NovaIntegracaoHTTP(cfg.KTMIntegracaoURL))
	}
	segredoPontoID := cfg.PontoIDSegredo
	if segredoPontoID == "" {
		segredoPontoID = "ponto-id|" + cfg.JWTSecret
	}
	servicoOrbit := orbit.NovoServico(orbit.NovoRepositorio(pool), orbit.NovoPontosAtivos(), segredoPontoID, integracoes...)
	leiturasHandler := NovoLeiturasHandler(servicoOrbit, repositorioUsuarios)
	redeHandler := NovoRedeHandler(servicoOrbit)
	redeWsHandler := ws.NovoRedeHandler(servicoOrbit, broadcasterRede)

	limitadorContas := deps.LimitadorContas
	if limitadorContas == nil {
		limitadorContas = NovoLimitador(5, 10*time.Minute)
	}
	r.Post("/auth/login", authHandler.Login)
	r.With(limitadorContas.Middleware).Post("/auth/cadastro", authHandler.Cadastro)
	r.With(limitadorContas.Middleware).Post("/auth/recuperar-senha", authHandler.RecuperarSenha)
	r.Post("/auth/redefinir-senha", authHandler.RedefinirSenha)
	r.Get("/termos", servirPagina("termos.html"))
	r.Get("/privacidade", servirPagina("privacidade.html"))

	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(cfg.JWTSecret))

		r.Get("/me", authHandler.Me)
		r.Patch("/me", authHandler.AtualizarMe)
		r.Delete("/me", authHandler.ExcluirMe)
		r.Put("/me/consentimentos/{tipo}", authHandler.DefinirConsentimento)
		r.Get("/admin/motoristas", cadastrosHandler.ListarMotoristas)
		r.Post("/admin/motoristas", cadastrosHandler.CriarMotorista)
		r.Get("/admin/veiculos", cadastrosHandler.ListarVeiculos)
		r.Post("/admin/veiculos", cadastrosHandler.CriarVeiculo)

		r.Get("/canais", canaisHandler.Listar)
		r.Get("/canais/explorar", canaisHandler.Explorar)
		r.Post("/canais", canaisHandler.Criar)
		r.Get("/canais/{id}", canaisHandler.Detalhar)
		r.Patch("/canais/{id}", canaisHandler.Editar)
		r.Delete("/canais/{id}", canaisHandler.Apagar)
		r.Post("/canais/{id}/convite", canaisHandler.NovoConvite)
		r.Get("/canais/{id}/participantes", canaisHandler.Participantes)
		r.Delete("/canais/{id}/participantes/{usuarioId}", canaisHandler.RemoverParticipante)
		r.Put("/canais/{id}/favorito", canaisHandler.Favoritar)
		r.Post("/convites/{codigo}", canaisHandler.EntrarPorConvite)
		r.Post("/canais/{id}/empresas", canaisHandler.LiberarEmpresa)
		r.Delete("/canais/{id}/empresas/{empresaId}", canaisHandler.RevogarEmpresa)
		r.Put("/canais/{id}/preferencia", canaisHandler.DefinirPreferencia)

		r.Post("/posicoes", posicoesHandler.Receber)
		r.Put("/posicao-atual", posicoesHandler.AtualizarAtual)
		r.Get("/empresas/{id}/posicoes-atuais", posicoesHandler.PosicoesAtuais)
		r.Get("/motoristas/{id}/trajeto", posicoesHandler.Trajeto)

		r.Post("/leituras", leiturasHandler.Receber)
		r.Post("/ponto/sinal", leiturasHandler.Sinal)
		r.Get("/me/contribuicao", leiturasHandler.Contribuicao)

		r.Route("/rede", func(r chi.Router) {
			r.Use(SomenteAdminKTM)
			r.Get("/orbits", redeHandler.Orbits)
			r.Get("/orbits/{id}/leituras", redeHandler.Leituras)
			r.Get("/pontos-fixos", redeHandler.PontosFixos)
			r.Post("/pontos-fixos", redeHandler.CriarPontoFixo)
			r.Delete("/pontos-fixos/{id}", redeHandler.ApagarPontoFixo)
			r.Post("/pontos-fixos/{id}/simular", redeHandler.Simular)
		})
		r.Get("/ws/rede", redeWsHandler.ServeHTTP)

		r.Get("/ws", wsHandler.ServeHTTP)
		r.Get("/ws/painel", painelWsHandler.ServeHTTP)
		r.Get("/central/canais", centralHandler.Listar)
		r.Get("/ws/central", centralHandler.ServeHTTP)
	})

	r.Handle("/painel/*", http.StripPrefix("/painel/", http.FileServerFS(painel.Arquivos)))

	return r
}

// servirPagina serve uma página estática embutida em painel/ num endereço
// curto e público (termos de uso e privacidade, linkados pelo app).
func servirPagina(nome string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, painel.Arquivos, nome)
	}
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
