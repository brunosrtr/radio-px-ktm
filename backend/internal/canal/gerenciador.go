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
// existir em memória. limiteParticipantes é sincronizado no hub a cada
// chamada, refletindo edições feitas via PATCH /canais/{id}.
func (g *Gerenciador) ObterOuCriar(canalID string, limiteParticipantes int) *Canal {
	g.mu.Lock()
	c, ok := g.canais[canalID]
	if !ok {
		c = NovoCanal(canalID, ComLimiteParticipantes(limiteParticipantes))
		g.canais[canalID] = c
	}
	g.mu.Unlock()

	if ok {
		c.DefinirLimiteParticipantes(limiteParticipantes)
	}
	return c
}

// Obter retorna o hub do canal informado sem criá-lo, útil para aplicar
// atualizações (ex.: preferência de silenciamento) só quando já há uma
// sessão ativa naquele canal.
func (g *Gerenciador) Obter(canalID string) (*Canal, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	c, ok := g.canais[canalID]
	return c, ok
}

// CanaisDoUsuario lista os canais em que o usuário está atualmente conectado
// — usado pelo verificador de geocerca (US3) a cada atualização de posição.
func (g *Gerenciador) CanaisDoUsuario(usuarioID string) []string {
	g.mu.Lock()
	defer g.mu.Unlock()

	var ids []string
	for id, c := range g.canais {
		if c.TemMembro(usuarioID) {
			ids = append(ids, id)
		}
	}
	return ids
}
