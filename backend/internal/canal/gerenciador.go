package canal

import "sync"

// Gerenciador mantém um hub por canal ativo, criado sob demanda na primeira
// entrada e reaproveitado enquanto houver referência a ele. O backend roda
// numa única instância na v1 (Princípio VI) — não há necessidade de
// compartilhar esse estado entre processos.
type Gerenciador struct {
	mu     sync.Mutex
	canais map[string]*Canal
}

// NovoGerenciador cria um registro vazio de canais.
func NovoGerenciador() *Gerenciador {
	return &Gerenciador{canais: make(map[string]*Canal)}
}

// ObterOuCriar retorna o hub do canal informado, criando-o se ainda não
// existir em memória.
func (g *Gerenciador) ObterOuCriar(canalID string) *Canal {
	g.mu.Lock()
	defer g.mu.Unlock()

	c, ok := g.canais[canalID]
	if !ok {
		c = NovoCanal(canalID)
		g.canais[canalID] = c
	}
	return c
}
