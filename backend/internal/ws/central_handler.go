package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/canal"
	"github.com/coder/websocket"
)

// CentralHandler acompanha canais autorizados e entrega sua FIFO de áudio
// apenas a administradores, sem registrar participação como motorista.
type CentralHandler struct {
	gerenciador *canal.Gerenciador
	repositorio *canal.Repositorio
}

func NovaCentralHandler(g *canal.Gerenciador, r *canal.Repositorio) *CentralHandler {
	return &CentralHandler{g, r}
}
func (h *CentralHandler) Listar(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsDoContexto(r.Context())
	if !ok || claims.Papel != auth.PapelAdmin {
		http.Error(w, "apenas administradores", http.StatusForbidden)
		return
	}
	canais, err := h.repositorio.ListarAutorizados(r.Context(), claims.EmpresaID)
	if err != nil {
		http.Error(w, "não foi possível consultar canais", 500)
		return
	}
	dados := make([]map[string]any, 0, len(canais))
	for _, c := range canais {
		estado := canal.EstadoCentral{Participantes: []canal.ParticipanteConectado{}, Fila: []canal.MensagemNaFila{}}
		if hub, ok := h.gerenciador.Obter(c.ID); ok {
			estado = hub.EstadoParaCentral()
		}
		dados = append(dados, map[string]any{"id": c.ID, "nome": c.Nome, "participantes": estado.Participantes, "fila": estado.Fila, "pode_editar": c.EmpresaID == claims.EmpresaID, "descricao": c.Descricao, "limite_participantes": c.LimiteParticipantes, "geocerca_ativa": c.GeocercaAtiva, "centro_latitude": c.CentroLatitude, "centro_longitude": c.CentroLongitude, "raio_metros": c.RaioMetros, "tipo_acesso": c.TipoAcesso})
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]any{"canais": dados})
}
func (h *CentralHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsDoContexto(r.Context())
	if !ok || claims.Papel != auth.PapelAdmin {
		http.Error(w, "apenas administradores", http.StatusForbidden)
		return
	}
	canalID := r.URL.Query().Get("canal_id")
	canais, err := h.repositorio.ListarAutorizados(r.Context(), claims.EmpresaID)
	if err != nil {
		http.Error(w, "não foi possível consultar canais", 500)
		return
	}
	var autorizado *canal.CanalDB
	for i := range canais {
		if canais[i].ID == canalID {
			autorizado = &canais[i]
			break
		}
	}
	if autorizado == nil {
		http.Error(w, "canal não autorizado", http.StatusForbidden)
		return
	}
	conn, err := websocket.Accept(w, r, opcoesAccept())
	if err != nil {
		return
	}
	defer conn.CloseNow()
	ctx, cancelar := context.WithCancel(r.Context())
	defer cancelar()
	hub := h.gerenciador.ObterOuCriar(canalID, autorizado.LimiteParticipantes)
	ouvinte, parar := hub.AssinarCentral()
	defer parar()
	go func() {
		defer cancelar()
		defer conn.CloseNow()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			var tipo websocket.MessageType
			var dados []byte
			select {
			case <-ctx.Done():
				return
			case <-ouvinte.Falha:
				return
			case <-ticker.C:
				if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
					return
				}
				// Revogação de acesso encerra a escuta mesmo sem atualizar a página.
				canalAtual, err := h.repositorio.ObterCanal(ctx, canalID)
				if err != nil {
					return
				}
				if canalAtual.EmpresaID != claims.EmpresaID {
					permitido, err := h.repositorio.EstaAutorizada(ctx, canalID, claims.EmpresaID)
					if err != nil || !permitido {
						return
					}
				}
				continue
			case evento := <-ouvinte.Eventos:
				if evento.Audio != nil {
					tipo = websocket.MessageBinary
					dados = evento.Audio
				} else {
					tipo = websocket.MessageText
					dados, err = json.Marshal(mensagemSaida{Tipo: evento.Tipo, Dados: evento.Dados})
					if err != nil {
						return
					}
				}
			}
			escrita, parar := context.WithTimeout(ctx, 10*time.Second)
			err := conn.Write(escrita, tipo, dados)
			parar()
			if err != nil {
				return
			}
		}
	}()
	for {
		if _, _, err := conn.Read(ctx); err != nil {
			return
		}
	}
}
