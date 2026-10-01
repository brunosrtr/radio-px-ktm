package canal

import (
	"fmt"
	"testing"
	"time"
)

func aguardarEvento(t *testing.T, membro *Membro, tipo string) Evento {
	t.Helper()
	limite := time.NewTimer(time.Second)
	defer limite.Stop()
	for {
		select {
		case evento := <-membro.Eventos:
			if evento.Tipo == tipo {
				return evento
			}
		case <-limite.C:
			t.Fatalf("evento %s não recebido", tipo)
		}
	}
}

func TestCorteEncerraERecusaNovosChunks(t *testing.T) {
	c := NovoCanal("corte", ComDuracaoMaximaTransmissao(25*time.Millisecond))
	defer c.Fechar()
	remetente, _, _ := c.Entrar("a", "A")
	ouvinte, _, _ := c.Entrar("b", "B")
	transmissao, err := c.SolicitarSlot("a", "A")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.ReceberChunk(transmissao.ID, []byte("voz")); err != nil {
		t.Fatal(err)
	}
	fim := aguardarEvento(t, remetente, "transmissao_encerrada")
	if fim.Dados["motivo"] != "limite" {
		t.Fatalf("motivo inesperado: %v", fim.Dados)
	}
	aguardarEvento(t, ouvinte, "fim_reproducao")
	if err := c.ReceberChunk(transmissao.ID, []byte("tardio")); err != ErrTransmissaoInexistente {
		t.Fatalf("chunk após corte aceito: %v", err)
	}
}

func TestSaidaDescartaTransmissaoIncompletaELiberaFila(t *testing.T) {
	c := NovoCanal("saida")
	defer c.Fechar()
	c.Entrar("a", "A")
	ouvinte, _, _ := c.Entrar("b", "B")
	primeira, _ := c.SolicitarSlot("a", "A")
	c.ReceberChunk(primeira.ID, []byte("incompleta"))
	aguardarEvento(t, ouvinte, "inicio_reproducao")
	c.Sair("a")
	if primeira.Chunks() != nil {
		t.Fatal("saída manteve buffer da transmissão incompleta")
	}
	if err := c.ReceberChunk(primeira.ID, []byte("tardio")); err != ErrTransmissaoInexistente {
		t.Fatalf("chunk após saída aceito: %v", err)
	}
	aguardarEvento(t, ouvinte, "fim_reproducao")
	estado := aguardarEvento(t, ouvinte, "estado_canal")
	if estado.Dados["tamanho_fila"] != 0 {
		t.Fatalf("fila não liberada: %v", estado.Dados)
	}
}

func TestRecusaSegundaCapturaDoMesmoMotorista(t *testing.T) {
	c := NovoCanal("duplicada")
	defer c.Fechar()
	c.Entrar("a", "A")
	if _, err := c.SolicitarSlot("a", "A"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.SolicitarSlot("a", "A"); err == nil {
		t.Fatal("segunda captura simultânea aceita")
	}
}

func TestFecharDescartaBuffersERecusaNovasTransmissoes(t *testing.T) {
	c := NovoCanal("fechado")
	c.Entrar("a", "A")
	transmissao, _ := c.SolicitarSlot("a", "A")
	c.ReceberChunk(transmissao.ID, []byte("voz"))
	c.Fechar()
	if transmissao.Chunks() != nil {
		t.Fatal("canal fechado manteve áudio")
	}
	if _, err := c.SolicitarSlot("a", "A"); err == nil {
		t.Fatal("canal fechado aceitou transmissão")
	}
}

func TestCancelamentoLiberaVagaSemEsperarFalaAtual(t *testing.T) {
	c := NovoCanal("cancelamento")
	defer c.Fechar()
	var ultima *Transmissao
	for i := 0; i < CapacidadeFila; i++ {
		var err error
		ultima, err = c.SolicitarSlot(fmt.Sprint(i), "Motorista")
		if err != nil {
			t.Fatal(err)
		}
	}
	c.ReceberChunk(ultima.ID, []byte("áudio ainda na fila"))
	c.CancelarTransmissao(ultima.ID)
	if ultima.Chunks() != nil {
		t.Fatal("cancelamento manteve áudio pendente")
	}
	if _, err := c.SolicitarSlot("novo", "Novo"); err != nil {
		t.Fatalf("vaga cancelada não foi liberada imediatamente: %v", err)
	}
}
