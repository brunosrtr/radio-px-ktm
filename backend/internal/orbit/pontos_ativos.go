package orbit

import (
	"sync"
	"time"
)

// JanelaPontosAtivos é o tempo em que um ponto de transmissão conta como
// ativo depois do último sinal (FR-018).
const JanelaPontosAtivos = 15 * time.Minute

// PontosAtivos guarda, só em memória, quando cada ponto_id deu sinal pela
// última vez. Não precisa sobreviver a reinício (Princípio VI).
type PontosAtivos struct {
	mu     sync.Mutex
	agora  func() time.Time
	vistos map[string]time.Time
}

// NovoPontosAtivos cria o registro vazio.
func NovoPontosAtivos() *PontosAtivos {
	return &PontosAtivos{agora: time.Now, vistos: map[string]time.Time{}}
}

// ComRelogio troca a fonte de tempo (testes).
func (p *PontosAtivos) ComRelogio(agora func() time.Time) *PontosAtivos {
	p.agora = agora
	return p
}

// Registrar marca o ponto como visto agora.
func (p *PontosAtivos) Registrar(pontoID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.vistos[pontoID] = p.agora()
}

// Contar devolve quantos pontos deram sinal dentro da janela e descarta os
// que já saíram dela.
func (p *PontosAtivos) Contar(janela time.Duration) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	limite := p.agora().Add(-janela)
	total := 0
	for id, visto := range p.vistos {
		if visto.After(limite) {
			total++
		} else {
			delete(p.vistos, id)
		}
	}
	return total
}
