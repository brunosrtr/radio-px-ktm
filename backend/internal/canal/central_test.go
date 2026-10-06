package canal

import (
	"testing"
	"time"
)

func TestCentralOuveFilaSemOcuparVaga(t *testing.T) {
	c := NovoCanal("central", ComLimiteParticipantes(1))
	defer c.Fechar()
	c.Entrar("a", "Motorista A")
	central, parar := c.AssinarCentral()
	defer parar()
	if c.Participantes() != 1 || len(c.EstadoParaCentral().Participantes) != 1 {
		t.Fatal("a central ocupou uma vaga de motorista")
	}
	if _, _, err := c.Entrar("b", "Motorista B"); err != ErrLimiteParticipantes {
		t.Fatal("limite de participantes alterado")
	}
	a, _ := c.SolicitarSlot("a", "Motorista A")
	inicio := aguardarEvento(t, central, "inicio_reproducao")
	if inicio.Dados["transmissao_id"] != a.ID {
		t.Fatal("início incorreto")
	}
	b, _ := c.SolicitarSlot("b", "Motorista B")
	c.ReceberChunk(b.ID, []byte("b"))
	c.FinalizarTransmissao(b.ID)
	estado := c.EstadoParaCentral()
	if len(estado.Fila) != 2 || estado.Fila[0].ID != a.ID || estado.Fila[1].ID != b.ID || estado.Fila[1].Estado != "aguardando" {
		t.Fatalf("snapshot fora de ordem: %+v", estado.Fila)
	}
	c.ReceberChunk(a.ID, []byte("a"))
	c.FinalizarTransmissao(a.ID)
	esperado := []struct{ tipo, audio string }{{"", "a"}, {"fim_reproducao", ""}, {"inicio_reproducao", ""}, {"", "b"}, {"fim_reproducao", ""}}
	for _, exp := range esperado {
		select {
		case evento := <-central.Eventos:
			if evento.Tipo != exp.tipo || string(evento.Audio) != exp.audio {
				t.Fatalf("FIFO alterada: %+v", evento)
			}
		case <-time.After(time.Second):
			t.Fatal("evento não chegou")
		}
	}
	if a.Chunks() != nil || b.Chunks() != nil {
		t.Fatal("buffer de áudio não foi descartado")
	}
}
func TestCentralNaoRecebeFalaPelaMetadeEParaSemReterPacotes(t *testing.T) {
	c := NovoCanal("parcial")
	defer c.Fechar()
	motorista, _, _ := c.Entrar("ouvinte", "Ouvinte")
	a, _ := c.SolicitarSlot("a", "Motorista A")
	aguardarEvento(t, motorista, "inicio_reproducao")
	central, parar := c.AssinarCentral()
	c.ReceberChunk(a.ID, []byte("parte final"))
	c.FinalizarTransmissao(a.ID)
	aguardarEvento(t, motorista, "fim_reproducao")
	if len(central.Eventos) != 0 {
		t.Fatal("entregou fala sem o início")
	}
	b, _ := c.SolicitarSlot("b", "Motorista B")
	aguardarEvento(t, central, "inicio_reproducao")
	c.ReceberChunk(b.ID, []byte("pendente"))
	parar()
	c.ReceberChunk(b.ID, []byte("após saída"))
	c.FinalizarTransmissao(b.ID)
	aguardarEvento(t, motorista, "fim_reproducao")
	if len(central.Eventos) != 0 {
		t.Fatal("central manteve áudio após parar")
	}
}
func TestCentralLentaEncerraSemDescartarEventosSilenciosamente(t *testing.T) {
	membro := novoMembro("central", "Central")
	membro.Eventos = make(chan Evento, 1)
	membro.Falha = make(chan struct{})
	membro.enviar(Evento{Audio: []byte("primeiro")})
	membro.enviar(Evento{Audio: []byte("segundo")})
	select {
	case <-membro.Falha:
	default:
		t.Fatal("estouro não sinalizou falha")
	}
}
