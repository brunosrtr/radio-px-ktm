package ws

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/coder/websocket"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/posicao"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/usuario"
)

// PainelHandler faz o upgrade da conexão do painel da empresa e transmite,
// em tempo real, cada atualização de posição publicada pelo Broadcaster —
// sem o admin precisar recarregar a página para ver o mapa se mover (US5).
type PainelHandler struct {
	broadcaster *posicao.Broadcaster
}

// NovoPainelHandler cria o handler de WebSocket do painel sobre o
// Broadcaster de posições informado.
func NovoPainelHandler(broadcaster *posicao.Broadcaster) *PainelHandler {
	return &PainelHandler{broadcaster: broadcaster}
}

// ServeHTTP só aceita administradores — o Broadcaster já isola por empresa,
// mas a checagem de papel aqui evita abrir a conexão para um motorista sem
// necessidade nenhuma (mesma postura de rejeitar antes de qualquer lógica de
// negócio de auth.Middleware).
func (h *PainelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsDoContexto(r.Context())
	if !ok {
		http.Error(w, `{"codigo":"token_invalido","mensagem":"token ausente ou inválido"}`, http.StatusUnauthorized)
		return
	}
	if claims.Papel != usuario.PapelAdmin {
		http.Error(w, `{"codigo":"sem_permissao","mensagem":"apenas administradores podem acompanhar o painel"}`, http.StatusForbidden)
		return
	}

	conn, err := websocket.Accept(w, r, opcoesAccept())
	if err != nil {
		return
	}
	defer conn.CloseNow()

	atualizacoes, cancelarAssinatura := h.broadcaster.Assinar(claims.EmpresaID)
	defer cancelarAssinatura()

	ctxEscrita, pararEscrita := context.WithCancel(context.Background())
	defer pararEscrita()
	go escreverAtualizacoes(ctxEscrita, conn, atualizacoes)

	// A leitura só existe para detectar o fechamento da conexão pelo
	// cliente — o painel não envia mensagens de controle.
	for {
		if _, _, err := conn.Read(r.Context()); err != nil {
			return
		}
	}
}

func escreverAtualizacoes(ctx context.Context, conn *websocket.Conn, atualizacoes <-chan posicao.MotoristaPosicaoAtual) {
	for {
		select {
		case atualizacao, aberto := <-atualizacoes:
			if !aberto {
				return
			}
			b, err := json.Marshal(mensagemSaida{Tipo: "posicao", Dados: map[string]any{
				"usuario_id":     atualizacao.UsuarioID,
				"nome":           atualizacao.Nome,
				"latitude":       atualizacao.Latitude,
				"longitude":      atualizacao.Longitude,
				"velocidade_kmh": atualizacao.VelocidadeKmh,
				"capturado_em":   atualizacao.CapturadoEm,
			}})
			if err != nil {
				continue
			}
			if err := conn.Write(ctx, websocket.MessageText, b); err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
