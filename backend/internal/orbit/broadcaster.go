package orbit

import "sync"

// Evento é uma mensagem para os painéis Rede Orbit, no envelope do WebSocket
// ({"tipo": ..., "dados": ...}) — contracts/painel-rede-ws.md.
type Evento struct {
	Tipo  string `json:"tipo"`
	Dados any    `json:"dados"`
}

// Broadcaster distribui eventos da Rede Orbit para os painéis conectados.
// Publicar nunca bloqueia: um painel lento perde o evento em vez de atrasar
// a ingestão de leituras.
type Broadcaster struct {
	mu         sync.Mutex
	assinantes map[chan Evento]struct{}
}

// NovoBroadcaster cria um Broadcaster vazio.
func NovoBroadcaster() *Broadcaster {
	return &Broadcaster{assinantes: map[chan Evento]struct{}{}}
}

// Assinar registra um painel. cancelar deve ser chamado quando a conexão
// terminar.
func (b *Broadcaster) Assinar() (eventos <-chan Evento, cancelar func()) {
	ch := make(chan Evento, 64)
	b.mu.Lock()
	b.assinantes[ch] = struct{}{}
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		if _, ok := b.assinantes[ch]; ok {
			delete(b.assinantes, ch)
			close(ch)
		}
		b.mu.Unlock()
	}
}

// Publicar entrega o evento a todos os painéis.
func (b *Broadcaster) Publicar(evento Evento) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.assinantes {
		select {
		case ch <- evento:
		default:
		}
	}
}
