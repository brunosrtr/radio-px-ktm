package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/orbit"
)

// IntervaloPontosAtivos é a frequência do evento pontos_ativos.
var IntervaloPontosAtivos = 30 * time.Second

// RedeHandler transmite ao painel Rede Orbit, ao vivo, as leituras e as
// mudanças de posição dos Orbits (contracts/painel-rede-ws.md). Restrito a
// admin_ktm (FR-021). Nenhum evento carrega dados de quem fez a leitura.
type RedeHandler struct {
	servico     *orbit.Servico
	broadcaster *orbit.Broadcaster
}

// NovoRedeHandler cria o handler.
func NovoRedeHandler(servico *orbit.Servico, broadcaster *orbit.Broadcaster) *RedeHandler {
	return &RedeHandler{servico: servico, broadcaster: broadcaster}
}

// ServeHTTP implementa GET /ws/rede.
func (h *RedeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsDoContexto(r.Context())
	if !ok {
		http.Error(w, `{"codigo":"token_invalido","mensagem":"token ausente ou inválido"}`, http.StatusUnauthorized)
		return
	}
	if claims.Papel != auth.PapelAdminKTM {
		http.Error(w, `{"codigo":"sem_permissao","mensagem":"o painel Rede Orbit é restrito à KTM"}`, http.StatusForbidden)
		return
	}

	// Assina antes de montar o estado inicial, para não perder leituras que
	// cheguem entre a consulta e o início da escuta.
	eventos, cancelar := h.broadcaster.Assinar()
	defer cancelar()

	orbits, err := h.servico.Repositorio().Orbits(r.Context())
	if err != nil {
		http.Error(w, `{"codigo":"erro_interno","mensagem":"erro interno"}`, http.StatusInternalServerError)
		return
	}
	pontos, err := h.servico.Repositorio().PontosFixos(r.Context())
	if err != nil {
		http.Error(w, `{"codigo":"erro_interno","mensagem":"erro interno"}`, http.StatusInternalServerError)
		return
	}

	conn, err := websocket.Accept(w, r, opcoesAccept())
	if err != nil {
		return
	}
	defer conn.CloseNow()

	ctx, parar := context.WithCancel(context.Background())
	defer parar()

	escrever := func(evento orbit.Evento) bool {
		b, err := json.Marshal(evento)
		if err != nil {
			return true
		}
		ctxEscrita, cancelarEscrita := context.WithTimeout(ctx, 5*time.Second)
		defer cancelarEscrita()
		return conn.Write(ctxEscrita, websocket.MessageText, b) == nil
	}
	contarAtivos := func() orbit.Evento {
		return orbit.Evento{Tipo: "pontos_ativos", Dados: map[string]int{"total": h.servico.PontosAtivos().Contar(orbit.JanelaPontosAtivos)}}
	}

	if !escrever(orbit.Evento{Tipo: "inicial", Dados: map[string]any{
		"orbits":        orbits,
		"pontos_fixos":  pontos,
		"pontos_ativos": h.servico.PontosAtivos().Contar(orbit.JanelaPontosAtivos),
	}}) {
		return
	}

	go func() {
		defer parar()
		ticker := time.NewTicker(IntervaloPontosAtivos)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case evento, aberto := <-eventos:
				if !aberto || !escrever(evento) {
					return
				}
				if evento.Tipo == "leitura" {
					// O contador sobe junto com a leitura, sem esperar o ticker.
					if !escrever(contarAtivos()) {
						return
					}
				}
			case <-ticker.C:
				if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
					return
				}
				if !escrever(contarAtivos()) {
					return
				}
			}
		}
	}()

	// A leitura só detecta o fechamento da conexão pelo painel.
	for {
		if _, _, err := conn.Read(ctx); err != nil {
			return
		}
	}
}
