// Package canal implementa o hub de canais de voz: uma fila FIFO em memória
// por canal (nunca em disco — Princípio I da constituição) que distribui
// cada transmissão aos membros conectados e descarta o áudio assim que a
// reprodução termina.
package canal

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sort"
	"sync"
	"time"
)

// CapacidadeFila é o número máximo de transmissões que podem estar
// enfileiradas (aguardando ou em reprodução) simultaneamente num canal
// (FR-06/FR-07).
const CapacidadeFila = 10

// DuracaoMaximaTransmissao é o tempo máximo de gravação de uma transmissão,
// aplicado pelo servidor independentemente do que o app envie (FR-04,
// research.md §3).
const DuracaoMaximaTransmissao = 90 * time.Second

var (
	// ErrFilaCheia indica que a fila do canal já tem 10 transmissões
	// pendentes de reprodução.
	ErrFilaCheia              = errors.New("canal: fila cheia")
	ErrCanalFechado           = errors.New("canal: fechado")
	ErrTransmissaoEmAndamento = errors.New("canal: transmissão em andamento")
	// ErrTransmissaoInexistente indica que a transmissão já foi finalizada,
	// cortada por limite de tempo, ou nunca existiu nesta conexão.
	ErrTransmissaoInexistente = errors.New("canal: transmissão inexistente ou já encerrada")
	// ErrLimiteParticipantes indica que o canal já está com o número máximo
	// de participantes simultâneos permitido (FR-023).
	ErrLimiteParticipantes = errors.New("canal: limite de participantes atingido")
)

// Evento é o envelope genérico enviado do hub para um membro — controle
// (Dados) ou um chunk de áudio (Audio), nunca os dois ao mesmo tempo.
type Evento struct {
	Tipo  string
	Dados map[string]any
	Audio []byte
}

// Canal é o hub em memória de um canal de voz: uma goroutine consumidora
// processa a fila estritamente em ordem de chegada (FIFO) e distribui cada
// transmissão aos membros não silenciados antes de zerar o buffer de chunks.
type Canal struct {
	ID string

	duracaoMaximaTransmissao time.Duration

	mu                  sync.Mutex
	limiteParticipantes int
	membros             map[string]*Membro
	ouvintes            map[string]*Membro
	emReproducao        string
	transmissoesAtivas  map[string]*Transmissao
	filaTransmissoes    []*Transmissao
	fechado             bool
	cond                *sync.Cond
}

// Opcao customiza a criação de um Canal — usada por testes para encurtar
// DuracaoMaximaTransmissao sem esperar 90s reais.
type Opcao func(*Canal)

// ComDuracaoMaximaTransmissao substitui o limite de 90s padrão.
func ComDuracaoMaximaTransmissao(d time.Duration) Opcao {
	return func(c *Canal) { c.duracaoMaximaTransmissao = d }
}

// ComLimiteParticipantes define o limite de participantes simultâneos
// (FR-022/FR-023). Zero (padrão) significa sem limite.
func ComLimiteParticipantes(n int) Opcao {
	return func(c *Canal) { c.limiteParticipantes = n }
}

// NovoCanal cria o hub de um canal e inicia sua goroutine consumidora.
func NovoCanal(id string, opcoes ...Opcao) *Canal {
	c := &Canal{
		ID:                       id,
		duracaoMaximaTransmissao: DuracaoMaximaTransmissao,
		membros:                  make(map[string]*Membro),
		ouvintes:                 make(map[string]*Membro),
		transmissoesAtivas:       make(map[string]*Transmissao),
	}
	c.cond = sync.NewCond(&c.mu)
	for _, opcao := range opcoes {
		opcao(c)
	}

	go c.consumir()

	return c
}

// DefinirLimiteParticipantes atualiza o limite de participantes simultâneos
// — usado para manter o hub em memória sincronizado após uma edição do canal
// (PATCH /canais/{id}).
func (c *Canal) DefinirLimiteParticipantes(n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.limiteParticipantes = n
}

// Entrar registra um novo membro no canal e retorna o total de participantes
// após a entrada. Recusa com ErrLimiteParticipantes quando o canal já está
// no limite (FR-023); geocerca (US3) é responsabilidade de camadas
// superiores.
func (c *Canal) Entrar(usuarioID, nome string) (membro *Membro, participantes int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, jaEsta := c.membros[usuarioID]; !jaEsta &&
		c.limiteParticipantes > 0 && len(c.membros) >= c.limiteParticipantes {
		return nil, 0, ErrLimiteParticipantes
	}

	m := novoMembro(usuarioID, nome)
	c.membros[usuarioID] = m
	c.publicarEstadoLocked()
	return m, len(c.membros), nil
}

// DefinirSilenciado atualiza a preferência de silenciamento do membro
// atualmente conectado, se houver (FR-019). Não é erro chamar para um
// usuário sem conexão ativa no canal — a preferência em si vive no banco e é
// aplicada de novo na próxima entrada.
func (c *Canal) DefinirSilenciado(usuarioID string, silenciado bool) {
	c.mu.Lock()
	m, ok := c.membros[usuarioID]
	c.mu.Unlock()
	if ok {
		m.DefinirSilenciado(silenciado)
	}
}

// Sair remove um membro do canal.
func (c *Canal) Sair(usuarioID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.membros, usuarioID)
	c.cancelarCapturasLocked(usuarioID)
	c.publicarEstadoLocked()
}

// TemMembro indica se o usuário está atualmente conectado ao canal.
func (c *Canal) TemMembro(usuarioID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.membros[usuarioID]
	return ok
}

// Remover força a saída de um membro do canal, notificando-o com o evento
// informado antes de descartar sua participação — usado por remoções
// iniciadas pelo servidor (geocerca, limite, canal desativado), diferente de
// Sair, que é a saída voluntária do próprio usuário.
func (c *Canal) Remover(usuarioID string, evento Evento) {
	c.mu.Lock()
	m, ok := c.membros[usuarioID]
	if ok {
		delete(c.membros, usuarioID)
		c.cancelarCapturasLocked(usuarioID)
		c.publicarEstadoLocked()
	}
	c.mu.Unlock()
	if ok {
		m.enviar(evento)
	}
}

// Participantes retorna o número de membros atualmente no canal.
func (c *Canal) Participantes() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.membros)
}

// TamanhoFila retorna quantas transmissões estão enfileiradas (aguardando ou
// em reprodução) no momento.
func (c *Canal) TamanhoFila() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.filaTransmissoes)
}

// SolicitarSlot reserva uma posição na fila para uma nova transmissão. É
// recusado com ErrFilaCheia quando já há 10 transmissões pendentes de
// reprodução (FR-06/FR-07). O corte automático em 90s (FR-04) começa a
// contar a partir desta chamada.
func (c *Canal) SolicitarSlot(remetenteID, remetenteNome string) (*Transmissao, error) {
	c.mu.Lock()
	if c.fechado {
		c.mu.Unlock()
		return nil, ErrCanalFechado
	}
	for _, ativa := range c.transmissoesAtivas {
		if ativa.RemetenteID == remetenteID {
			c.mu.Unlock()
			return nil, ErrTransmissaoEmAndamento
		}
	}
	if len(c.filaTransmissoes) >= CapacidadeFila {
		c.mu.Unlock()
		return nil, ErrFilaCheia
	}

	t := novaTransmissao(novoID(), remetenteID, remetenteNome)
	c.filaTransmissoes = append(c.filaTransmissoes, t)
	c.transmissoesAtivas[t.ID] = t
	duracaoMaxima := c.duracaoMaximaTransmissao
	c.publicarEstadoLocked()
	c.mu.Unlock()

	c.cond.Signal()
	go c.cortarAposLimite(t, duracaoMaxima)

	return t, nil
}

// ReceberChunk anexa um chunk de áudio recebido em frames binários à
// transmissão ativa correspondente.
func (c *Canal) ReceberChunk(transmissaoID string, chunk []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	t, ok := c.transmissoesAtivas[transmissaoID]
	if !ok || !t.receberChunk(chunk) {
		return ErrTransmissaoInexistente
	}
	return nil
}

// FinalizarTransmissao encerra a captura, preservando somente o áudio que
// ainda aguarda sua primeira reprodução na fila.
func (c *Canal) FinalizarTransmissao(transmissaoID string) error {
	return c.finalizarTransmissao(transmissaoID, "concluida")
}

func (c *Canal) finalizarTransmissao(transmissaoID, motivo string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	t, ok := c.transmissoesAtivas[transmissaoID]
	if !ok {
		return ErrTransmissaoInexistente
	}
	delete(c.transmissoesAtivas, transmissaoID)
	t.finalizar()
	if m := c.membros[t.RemetenteID]; m != nil {
		m.enviar(Evento{Tipo: "transmissao_encerrada", Dados: map[string]any{
			"transmissao_id": t.ID, "motivo": motivo,
		}})
	}
	return nil
}

// CancelarTransmissao descarta a captura sem reproduzir o restante.
func (c *Canal) CancelarTransmissao(transmissaoID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t := c.transmissoesAtivas[transmissaoID]; t != nil {
		c.descartarCapturaLocked(t)
	}
}

// Chamado com mu adquirido. Transmissões já concluídas continuam na FIFO;
// somente capturas incompletas são descartadas na saída/desconexão.
func (c *Canal) cancelarCapturasLocked(usuarioID string) {
	for _, t := range c.transmissoesAtivas {
		if t.RemetenteID == usuarioID {
			c.descartarCapturaLocked(t)
		}
	}
}

func (c *Canal) descartarCapturaLocked(t *Transmissao) {
	delete(c.transmissoesAtivas, t.ID)
	t.cancelar()
	for i, pendente := range c.filaTransmissoes {
		if pendente == t {
			copy(c.filaTransmissoes[i:], c.filaTransmissoes[i+1:])
			ultimo := len(c.filaTransmissoes) - 1
			c.filaTransmissoes[ultimo] = nil
			c.filaTransmissoes = c.filaTransmissoes[:ultimo]
			break
		}
	}
	c.publicarEstadoLocked()
}

// publicarEstadoLocked ordena as atualizações junto das mutações do hub.
func (c *Canal) publicarEstadoLocked() {
	evento := Evento{Tipo: "estado_canal", Dados: map[string]any{
		"participantes": len(c.membros), "tamanho_fila": len(c.filaTransmissoes),
	}}
	for _, m := range c.membros {
		m.enviar(evento)
	}
}

// Fechar cancela capturas e libera a goroutine mesmo durante uma fala.
func (c *Canal) Fechar() {
	c.mu.Lock()
	c.fechado = true
	for _, t := range c.filaTransmissoes {
		t.cancelar()
	}
	clear(c.transmissoesAtivas)
	for _, ouvinte := range c.ouvintes {
		ouvinte.falhaUmaVez.Do(func() { close(ouvinte.Falha) })
	}
	c.filaTransmissoes = nil
	c.mu.Unlock()
	c.cond.Broadcast()
}

func (c *Canal) cortarAposLimite(t *Transmissao, duracaoMaxima time.Duration) {
	timer := time.NewTimer(duracaoMaxima)
	defer timer.Stop()
	select {
	case <-timer.C:
		_ = c.finalizarTransmissao(t.ID, "limite")
	case <-t.finalizadaCh:
	}
}

// consumir processa a fila estritamente em ordem de chegada: só avança para
// a próxima transmissão depois que a atual terminou de ser reproduzida a
// todos os destinatários (FR-05). reproduzir já entrega cada chunk assim que
// ele chega do remetente — não espera a gravação inteira terminar — então
// uma transmissão começa a tocar para quem está ouvindo assim que vira a
// cabeça da fila, ao vivo, como um rádio de verdade.
func (c *Canal) consumir() {
	for {
		c.mu.Lock()
		for len(c.filaTransmissoes) == 0 && !c.fechado {
			c.cond.Wait()
		}
		if c.fechado {
			c.mu.Unlock()
			return
		}
		t := c.filaTransmissoes[0]
		c.emReproducao = t.ID
		c.mu.Unlock()

		c.reproduzir(t)

		c.mu.Lock()
		c.emReproducao = ""
		if len(c.filaTransmissoes) > 0 && c.filaTransmissoes[0] == t {
			c.filaTransmissoes[0] = nil
			c.filaTransmissoes = c.filaTransmissoes[1:]
		}
		if !c.fechado {
			c.publicarEstadoLocked()
		}
		c.mu.Unlock()
	}
}

func (c *Canal) reproduzir(t *Transmissao) {
	if t.foiCancelada() {
		return
	}
	c.mu.Lock()
	destinatarios := make([]*Membro, 0, len(c.membros))
	for _, m := range c.membros {
		if m.UsuarioID == t.RemetenteID || m.Silenciado() {
			continue
		}
		destinatarios = append(destinatarios, m)
	}
	for _, m := range c.ouvintes {
		destinatarios = append(destinatarios, m)
	}
	c.mu.Unlock()

	inicio := Evento{Tipo: "inicio_reproducao", Dados: map[string]any{
		"transmissao_id": t.ID,
		"remetente_nome": t.RemetenteNome,
	}}
	for _, m := range destinatarios {
		c.enviarDestinatario(m, inicio)
	}

	for lido := 0; ; {
		chunk, proximoLido, ok := t.proximoChunk(lido)
		if !ok {
			break
		}
		lido = proximoLido
		for _, m := range destinatarios {
			c.mu.Lock()
			if (c.membros[m.UsuarioID] == m || c.ouvintes[m.UsuarioID] == m) && !m.Silenciado() && !t.foiCancelada() {
				m.enviar(Evento{Audio: chunk})
			}
			c.mu.Unlock()
		}
	}

	t.zerarBuffer()
	fim := Evento{Tipo: "fim_reproducao", Dados: map[string]any{"transmissao_id": t.ID}}
	for _, m := range destinatarios {
		c.enviarDestinatario(m, fim)
	}

}

func (c *Canal) enviarDestinatario(m *Membro, evento Evento) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.membros[m.UsuarioID] == m || c.ouvintes[m.UsuarioID] == m {
		m.enviar(evento)
	}
}

func novoID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ParticipanteConectado informa presença real no hub, independente do GPS.
type ParticipanteConectado struct {
	UsuarioID string `json:"usuario_id"`
	Nome      string `json:"nome"`
}
type MensagemNaFila struct {
	ID            string `json:"id"`
	RemetenteNome string `json:"remetente_nome"`
	Estado        string `json:"estado"`
}
type EstadoCentral struct {
	Participantes []ParticipanteConectado `json:"participantes"`
	Fila          []MensagemNaFila        `json:"fila"`
}

func (c *Canal) EstadoParaCentral() EstadoCentral {
	c.mu.Lock()
	defer c.mu.Unlock()
	estado := EstadoCentral{Participantes: []ParticipanteConectado{}, Fila: []MensagemNaFila{}}
	for _, m := range c.membros {
		estado.Participantes = append(estado.Participantes, ParticipanteConectado{m.UsuarioID, m.Nome})
	}
	sort.Slice(estado.Participantes, func(i, j int) bool { return estado.Participantes[i].Nome < estado.Participantes[j].Nome })
	for _, t := range c.filaTransmissoes {
		situacao := "aguardando"
		if _, ok := c.transmissoesAtivas[t.ID]; ok {
			situacao = "gravando"
		}
		if t.ID == c.emReproducao {
			situacao = "em reprodução"
		}
		estado.Fila = append(estado.Fila, MensagemNaFila{t.ID, t.RemetenteNome, situacao})
	}
	return estado
}

// AssinarCentral escuta a FIFO sem ocupar uma vaga de motorista. Uma fala
// já iniciada não é enviada parcialmente; a escuta começa na próxima fala.
func (c *Canal) AssinarCentral() (*Membro, func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := novoID()
	m := novoMembro(id, "Central")
	m.Eventos = make(chan Evento, 8192)
	m.Falha = make(chan struct{})
	c.ouvintes[id] = m
	return m, func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		delete(c.ouvintes, id)
		// Libera os pacotes ainda não encaminhados quando a aba para de ouvir.
		for {
			select {
			case <-m.Eventos:
			default:
				return
			}
		}
	}
}
