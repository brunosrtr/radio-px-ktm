package canal

import (
	"sync"
	"time"
)

// Transmissao representa uma mensagem de voz em trânsito. Existe somente em
// memória — os chunks nunca tocam disco, banco ou log (Princípio I) — e seu
// buffer é zerado assim que a reprodução aos destinatários termina.
//
// Os chunks são entregues aos ouvintes em tempo real, conforme chegam do
// remetente (não só depois que a gravação inteira termina): proximoChunk
// bloqueia até haver um chunk ainda não lido ou a transmissão finalizar.
type Transmissao struct {
	ID            string
	RemetenteID   string
	RemetenteNome string
	IniciadaEm    time.Time

	mu           sync.Mutex
	cond         *sync.Cond
	chunks       [][]byte
	finalizada   bool
	cancelada    bool
	finalizadaCh chan struct{}
}

func novaTransmissao(id, remetenteID, remetenteNome string) *Transmissao {
	t := &Transmissao{
		ID:            id,
		RemetenteID:   remetenteID,
		RemetenteNome: remetenteNome,
		IniciadaEm:    time.Now(),
		finalizadaCh:  make(chan struct{}),
	}
	t.cond = sync.NewCond(&t.mu)
	return t
}

func (t *Transmissao) receberChunk(chunk []byte) bool {
	t.mu.Lock()
	if t.finalizada {
		t.mu.Unlock()
		return false
	}
	t.chunks = append(t.chunks, chunk)
	t.mu.Unlock()
	t.cond.Broadcast()
	return true
}

// finalizar marca a transmissão como encerrada: não chegam mais chunks
// novos, e proximoChunk para de bloquear assim que os já recebidos forem
// entregues. É seguro chamar mais de uma vez (corte automático por tempo e
// finalizar_transmissao podem competir).
func (t *Transmissao) finalizar() {
	t.mu.Lock()
	if t.finalizada {
		t.mu.Unlock()
		return
	}
	t.finalizada = true
	t.mu.Unlock()
	close(t.finalizadaCh)
	t.cond.Broadcast()
}

// proximoChunk bloqueia até o chunk de índice lido estar disponível ou a
// transmissão finalizar sem mais chunks pendentes — é o que permite a
// goroutine consumidora do hub transmitir ao vivo em vez de esperar a
// gravação inteira antes de tocar qualquer coisa.
func (t *Transmissao) proximoChunk(lido int) (chunk []byte, proximoLido int, ok bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for lido >= len(t.chunks) && !t.finalizada {
		t.cond.Wait()
	}
	if lido < len(t.chunks) {
		return t.chunks[lido], lido + 1, true
	}
	return nil, lido, false
}

// Chunks retorna os chunks acumulados até o momento (usado por testes para
// verificar que o buffer foi zerado após a reprodução).
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

// cancelar descarta uma captura incompleta e acorda o consumidor sem
// reproduzir os chunks restantes. Não altera buffers já entregues à rede.
func (t *Transmissao) cancelar() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cancelada = true
	t.chunks = nil
	if !t.finalizada {
		t.finalizada = true
		close(t.finalizadaCh)
	}
	t.cond.Broadcast()
}

func (t *Transmissao) foiCancelada() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cancelada
}
