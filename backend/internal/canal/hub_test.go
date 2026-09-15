package canal

import (
	"testing"
	"time"
)

func TestFilaAceitaAte10MensagensERecusaA11(t *testing.T) {
	c := NovoCanal("canal-teste", ComDuracaoMaximaTransmissao(time.Minute))
	defer c.Fechar()

	for i := 0; i < CapacidadeFila; i++ {
		if _, err := c.SolicitarSlot("remetente", "Remetente"); err != nil {
			t.Fatalf("solicitação %d deveria ter sido aceita, veio erro: %v", i+1, err)
		}
	}

	// Nenhuma das 10 transmissões foi finalizada, então a goroutine
	// consumidora está bloqueada na primeira — a fila permanece cheia.
	if _, err := c.SolicitarSlot("remetente", "Remetente"); err != ErrFilaCheia {
		t.Fatalf("11ª solicitação deveria ser recusada com ErrFilaCheia, veio: %v", err)
	}
}

func TestFilaReproduzEmOrdemDeChegadaEZeraBuffer(t *testing.T) {
	c := NovoCanal("canal-teste", ComDuracaoMaximaTransmissao(time.Minute))
	defer c.Fechar()

	ouvinte, _ := c.Entrar("ouvinte", "Ouvinte")

	t1, err := c.SolicitarSlot("motorista-a", "Motorista A")
	if err != nil {
		t.Fatalf("erro ao solicitar slot para t1: %v", err)
	}
	_ = c.ReceberChunk(t1.ID, []byte("a1"))
	_ = c.ReceberChunk(t1.ID, []byte("a2"))
	if err := c.FinalizarTransmissao(t1.ID); err != nil {
		t.Fatalf("erro ao finalizar t1: %v", err)
	}

	t2, err := c.SolicitarSlot("motorista-b", "Motorista B")
	if err != nil {
		t.Fatalf("erro ao solicitar slot para t2: %v", err)
	}
	_ = c.ReceberChunk(t2.ID, []byte("b1"))
	if err := c.FinalizarTransmissao(t2.ID); err != nil {
		t.Fatalf("erro ao finalizar t2: %v", err)
	}

	esperado := []struct {
		tipo          string
		transmissaoID string
		audio         string
	}{
		{tipo: "inicio_reproducao", transmissaoID: t1.ID},
		{audio: "a1"},
		{audio: "a2"},
		{tipo: "fim_reproducao", transmissaoID: t1.ID},
		{tipo: "inicio_reproducao", transmissaoID: t2.ID},
		{audio: "b1"},
		{tipo: "fim_reproducao", transmissaoID: t2.ID},
	}

	for i, exp := range esperado {
		select {
		case evento := <-ouvinte.Eventos:
			if exp.audio != "" {
				if string(evento.Audio) != exp.audio {
					t.Fatalf("evento %d: esperava áudio %q, veio %q", i, exp.audio, evento.Audio)
				}
				continue
			}
			if evento.Tipo != exp.tipo {
				t.Fatalf("evento %d: esperava tipo %q, veio %q", i, exp.tipo, evento.Tipo)
			}
			if evento.Dados["transmissao_id"] != exp.transmissaoID {
				t.Fatalf("evento %d: esperava transmissao_id %q, veio %q", i, exp.transmissaoID, evento.Dados["transmissao_id"])
			}
		case <-time.After(time.Second):
			t.Fatalf("evento %d (%+v) não chegou a tempo", i, exp)
		}
	}

	if chunks := t1.Chunks(); chunks != nil {
		t.Fatalf("buffer de t1 deveria estar zerado após a reprodução, veio: %v", chunks)
	}
	if chunks := t2.Chunks(); chunks != nil {
		t.Fatalf("buffer de t2 deveria estar zerado após a reprodução, veio: %v", chunks)
	}
}

func TestTransmissaoECortadaAutomaticamenteApos90Segundos(t *testing.T) {
	duracaoCurta := 30 * time.Millisecond
	c := NovoCanal("canal-teste", ComDuracaoMaximaTransmissao(duracaoCurta))
	defer c.Fechar()

	ouvinte, _ := c.Entrar("ouvinte", "Ouvinte")

	t1, err := c.SolicitarSlot("motorista-a", "Motorista A")
	if err != nil {
		t.Fatalf("erro ao solicitar slot: %v", err)
	}
	_ = c.ReceberChunk(t1.ID, []byte("a1"))
	// Nenhum finalizar_transmissao é enviado — o corte deve ser automático.

	select {
	case evento := <-ouvinte.Eventos:
		if evento.Tipo != "inicio_reproducao" {
			t.Fatalf("esperava inicio_reproducao, veio %q", evento.Tipo)
		}
	case <-time.After(time.Second):
		t.Fatal("transmissão não foi cortada e reproduzida automaticamente após o limite de tempo")
	}
}
