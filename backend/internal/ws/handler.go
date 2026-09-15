// Package ws implementa o upgrade e o roteamento de eventos do protocolo
// WebSocket (contracts/websocket-protocol.md): controle em frames de texto
// JSON, áudio em frames binários associados à transmissão ativa da conexão.
package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/coder/websocket"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/canal"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/usuario"
)

// mensagemEntrada é o envelope das mensagens de controle recebidas do
// cliente: {"tipo": "...", "dados": {...}}.
type mensagemEntrada struct {
	Tipo  string          `json:"tipo"`
	Dados json.RawMessage `json:"dados"`
}

type mensagemSaida struct {
	Tipo  string         `json:"tipo"`
	Dados map[string]any `json:"dados"`
}

// Handler faz o upgrade da conexão HTTP para WebSocket e roteia os eventos
// do protocolo para o Gerenciador de canais.
type Handler struct {
	gerenciador *canal.Gerenciador
	servico     *canal.Servico
	repositorio *canal.Repositorio
	usuarios    *usuario.Repositorio
}

// NovoHandler cria o handler de WebSocket sobre as dependências informadas.
func NovoHandler(gerenciador *canal.Gerenciador, servico *canal.Servico, repositorio *canal.Repositorio, usuarios *usuario.Repositorio) *Handler {
	return &Handler{gerenciador: gerenciador, servico: servico, repositorio: repositorio, usuarios: usuarios}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsDoContexto(r.Context())
	if !ok {
		http.Error(w, `{"codigo":"token_invalido","mensagem":"token ausente ou inválido"}`, http.StatusUnauthorized)
		return
	}

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("ws: erro no upgrade: %v", err)
		return
	}
	defer conn.CloseNow()

	ctx := r.Context()

	u, err := h.usuarios.ObterPorID(ctx, claims.UsuarioID)
	if err != nil {
		_ = conn.Close(websocket.StatusInternalError, "erro ao carregar usuário")
		return
	}

	s := &sessao{handler: h, conn: conn, usuarioID: claims.UsuarioID, empresaID: claims.EmpresaID, nome: u.Nome}
	defer s.encerrar()

	for {
		tipo, dados, err := conn.Read(ctx)
		if err != nil {
			return
		}

		switch tipo {
		case websocket.MessageBinary:
			s.receberChunkAudio(dados)
		case websocket.MessageText:
			s.processarMensagem(ctx, dados)
		}
	}
}

// sessao é o estado de uma conexão WebSocket: o usuário autenticado, o canal
// em que está (no máximo um por vez) e a transmissão que está gravando, se
// houver.
type sessao struct {
	handler   *Handler
	conn      *websocket.Conn
	usuarioID string
	empresaID string
	nome      string

	escritaMu sync.Mutex

	mu                  sync.Mutex
	canalAtual          *canal.Canal
	canalID             string
	membro              *canal.Membro
	transmissaoID       string
	pararEncaminhamento context.CancelFunc
}

func (s *sessao) processarMensagem(ctx context.Context, dados []byte) {
	var msg mensagemEntrada
	if err := json.Unmarshal(dados, &msg); err != nil {
		s.enviarErro(ctx, "mensagem_invalida", "corpo da mensagem não é JSON válido")
		return
	}

	switch msg.Tipo {
	case "entrar_canal":
		var corpo struct {
			CanalID string `json:"canal_id"`
		}
		if err := json.Unmarshal(msg.Dados, &corpo); err != nil || corpo.CanalID == "" {
			s.enviarErro(ctx, "mensagem_invalida", "canal_id é obrigatório")
			return
		}
		s.entrarCanal(ctx, corpo.CanalID)

	case "sair_canal":
		s.sairCanalAtual("manual")

	case "solicitar_slot":
		s.solicitarSlot(ctx)

	case "finalizar_transmissao":
		var corpo struct {
			TransmissaoID string `json:"transmissao_id"`
		}
		_ = json.Unmarshal(msg.Dados, &corpo)
		s.finalizarTransmissao(corpo.TransmissaoID)

	case "ping":
		// Mantém a conexão viva; o contrato não exige resposta.

	default:
		s.enviarErro(ctx, "tipo_desconhecido", fmt.Sprintf("tipo de mensagem desconhecido: %s", msg.Tipo))
	}
}

func (s *sessao) entrarCanal(ctx context.Context, canalID string) {
	s.sairCanalAtual("manual")

	canalDB, err := s.handler.servico.ObterParaEntrada(ctx, canalID, s.empresaID)
	if err != nil {
		switch {
		case errors.Is(err, canal.ErrNaoEncontrado):
			s.enviarErro(ctx, "nao_encontrado", "canal não encontrado")
		case errors.Is(err, canal.ErrSemPermissao):
			s.enviarErro(ctx, "sem_permissao", "empresa não autorizada a acessar este canal")
		default:
			log.Printf("ws: erro ao buscar canal para entrada: %v", err)
			s.enviarErro(ctx, "erro_interno", "não foi possível validar o canal")
		}
		return
	}

	c := s.handler.gerenciador.ObterOuCriar(canalID, canalDB.LimiteParticipantes)

	membro, total, err := c.Entrar(s.usuarioID, s.nome)
	if err != nil {
		s.enviarErro(ctx, "limite_atingido", "canal já está com o número máximo de participantes")
		return
	}

	if err := s.handler.repositorio.RegistrarEntrada(ctx, canalID, s.usuarioID); err != nil {
		log.Printf("ws: erro ao registrar entrada: %v", err)
		c.Sair(s.usuarioID)
		s.enviarErro(ctx, "erro_interno", "não foi possível registrar entrada no canal")
		return
	}

	if silenciado, err := s.handler.repositorio.ObterPreferencia(ctx, s.usuarioID, canalID); err == nil && silenciado {
		membro.DefinirSilenciado(true)
	}

	ctxMembro, cancelar := context.WithCancel(context.Background())

	s.mu.Lock()
	s.canalAtual = c
	s.canalID = canalID
	s.membro = membro
	s.pararEncaminhamento = cancelar
	s.mu.Unlock()

	go s.encaminharEventos(ctxMembro, membro)

	s.enviarEvento(ctx, canal.Evento{Tipo: "canal_entrado", Dados: map[string]any{
		"canal_id":      canalID,
		"participantes": total,
		"tamanho_fila":  c.TamanhoFila(),
	}})
}

func (s *sessao) sairCanalAtual(motivo string) {
	s.mu.Lock()
	c := s.canalAtual
	canalID := s.canalID
	cancelar := s.pararEncaminhamento
	s.canalAtual = nil
	s.canalID = ""
	s.membro = nil
	s.transmissaoID = ""
	s.pararEncaminhamento = nil
	s.mu.Unlock()

	if c == nil {
		return
	}

	if cancelar != nil {
		cancelar()
	}
	c.Sair(s.usuarioID)

	if err := s.handler.repositorio.RegistrarSaida(context.Background(), canalID, s.usuarioID, motivo); err != nil {
		log.Printf("ws: erro ao registrar saída: %v", err)
	}
}

func (s *sessao) solicitarSlot(ctx context.Context) {
	s.mu.Lock()
	c := s.canalAtual
	s.mu.Unlock()

	if c == nil {
		s.enviarErro(ctx, "sem_permissao", "é preciso entrar em um canal antes de solicitar um slot")
		return
	}

	t, err := c.SolicitarSlot(s.usuarioID, s.nome)
	if err != nil {
		s.enviarEvento(ctx, canal.Evento{Tipo: "slot_negado", Dados: map[string]any{"motivo": "fila_cheia"}})
		return
	}

	s.mu.Lock()
	s.transmissaoID = t.ID
	s.mu.Unlock()

	s.enviarEvento(ctx, canal.Evento{Tipo: "slot_concedido", Dados: map[string]any{
		"transmissao_id":    t.ID,
		"duracao_maxima_ms": canal.DuracaoMaximaTransmissao.Milliseconds(),
	}})
}

func (s *sessao) receberChunkAudio(dados []byte) {
	s.mu.Lock()
	c := s.canalAtual
	transmissaoID := s.transmissaoID
	s.mu.Unlock()

	if c == nil || transmissaoID == "" {
		return
	}
	_ = c.ReceberChunk(transmissaoID, dados)
}

func (s *sessao) finalizarTransmissao(transmissaoID string) {
	s.mu.Lock()
	c := s.canalAtual
	esperada := s.transmissaoID
	s.mu.Unlock()

	if c == nil || transmissaoID == "" || transmissaoID != esperada {
		return
	}

	if err := c.FinalizarTransmissao(transmissaoID); err == nil {
		s.mu.Lock()
		s.transmissaoID = ""
		s.mu.Unlock()
	}
}

func (s *sessao) encaminharEventos(ctx context.Context, membro *canal.Membro) {
	for {
		select {
		case e := <-membro.Eventos:
			if e.Audio != nil {
				s.escrever(ctx, websocket.MessageBinary, e.Audio)
				continue
			}
			s.enviarEvento(ctx, e)
		case <-ctx.Done():
			return
		}
	}
}

func (s *sessao) enviarEvento(ctx context.Context, e canal.Evento) {
	b, err := json.Marshal(mensagemSaida{Tipo: e.Tipo, Dados: e.Dados})
	if err != nil {
		log.Printf("ws: erro ao serializar evento %q: %v", e.Tipo, err)
		return
	}
	s.escrever(ctx, websocket.MessageText, b)
}

func (s *sessao) enviarErro(ctx context.Context, codigo, mensagem string) {
	s.enviarEvento(ctx, canal.Evento{Tipo: "erro", Dados: map[string]any{
		"codigo":   codigo,
		"mensagem": mensagem,
	}})
}

// escrever serializa as escritas na conexão: coder/websocket não permite
// mais de um escritor concorrente, e a sessão escreve tanto a partir da
// goroutine de leitura (respostas síncronas) quanto da de encaminhamento de
// eventos do hub.
func (s *sessao) escrever(ctx context.Context, tipo websocket.MessageType, dados []byte) {
	s.escritaMu.Lock()
	defer s.escritaMu.Unlock()
	if err := s.conn.Write(ctx, tipo, dados); err != nil {
		log.Printf("ws: erro ao escrever na conexão: %v", err)
	}
}

func (s *sessao) encerrar() {
	s.sairCanalAtual("desconexao")
}
