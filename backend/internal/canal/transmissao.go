package canal

import (
	"sync"
	"time"
)

// Transmissao representa uma mensagem de voz em trânsito. Existe somente em
// memória — os chunks nunca tocam disco, banco ou log (Princípio I) — e seu
// buffer é zerado assim que a reprodução aos destinatários termina.
type Transmissao struct {
	ID            string
	RemetenteID   string
	RemetenteNome string
	IniciadaEm    time.Time

	mu       sync.Mutex
	chunks   [][]byte
	pronta   bool
	prontaCh chan struct{}
}

func novaTransmissao(id, remetenteID, remetenteNome string) *Transmissao {
	return &Transmissao{
		ID:            id,
		RemetenteID:   remetenteID,
		RemetenteNome: remetenteNome,
		IniciadaEm:    time.Now(),
		prontaCh:      make(chan struct{}),
	}
}

func (t *Transmissao) receberChunk(chunk []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pronta {
		return
	}
	t.chunks = append(t.chunks, chunk)
}

// finalizar marca a transmissão como pronta para reprodução. É seguro chamar
// mais de uma vez (corte automático por tempo e finalizar_transmissao podem
// competir).
func (t *Transmissao) finalizar() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pronta {
		return
	}
	t.pronta = true
	close(t.prontaCh)
}

// Chunks retorna os chunks acumulados até o momento.
func (t *Transmissao) Chunks() [][]byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.chunks
}

func (t *Transmissao) zerarBuffer() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.chunks = nil
}
