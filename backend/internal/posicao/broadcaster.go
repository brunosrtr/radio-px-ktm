package posicao

import "sync"

// Broadcaster distribui atualizações de posição para os painéis de empresa
// conectados via WebSocket em tempo real — um admin só recebe as
// atualizações da própria empresa (mesmo isolamento de FR-026).
type Broadcaster struct {
	mu         sync.Mutex
	assinantes map[string]map[chan MotoristaPosicaoAtual]struct{}
}

// NovoBroadcaster cria um Broadcaster vazio.
func NovoBroadcaster() *Broadcaster {
	return &Broadcaster{assinantes: make(map[string]map[chan MotoristaPosicaoAtual]struct{})}
}

// Assinar registra um novo ouvinte para as atualizações de posição da
// empresa informada. Cancelar fecha o canal e remove a assinatura — deve ser
// chamado sempre que a conexão WebSocket do painel terminar.
func (b *Broadcaster) Assinar(empresaID string) (canal <-chan MotoristaPosicaoAtual, cancelar func()) {
	ch := make(chan MotoristaPosicaoAtual, 16)

	b.mu.Lock()
	if b.assinantes[empresaID] == nil {
		b.assinantes[empresaID] = make(map[chan MotoristaPosicaoAtual]struct{})
	}
	b.assinantes[empresaID][ch] = struct{}{}
	b.mu.Unlock()

	return ch, func() {
		b.mu.Lock()
		delete(b.assinantes[empresaID], ch)
		b.mu.Unlock()
		close(ch)
	}
}

// Publicar entrega a posição atualizada a todos os painéis assinantes da
// empresa. Não bloqueia o chamador: um assinante lento perde a atualização
// mais antiga em vez de travar a ingestão de posições (mesma filosofia de
// Membro.enviar no hub de canais).
func (b *Broadcaster) Publicar(empresaID string, atualizacao MotoristaPosicaoAtual) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for ch := range b.assinantes[empresaID] {
		select {
		case ch <- atualizacao:
		default:
		}
	}
}
