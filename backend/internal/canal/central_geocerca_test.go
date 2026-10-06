package canal

import (
	"context"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/posicao"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage/testutil"
	"testing"
	"time"
)

func TestEdicaoDoRaioRemoveMotoristaJaConectado(t *testing.T) {
	pool := testutil.AbrirPool(t)
	ctx := context.Background()
	empresa := testutil.CriarEmpresa(t, pool)
	motorista := testutil.CriarUsuario(t, pool, empresa, "motorista")
	repositorio := NovoRepositorio(pool)
	gerenciador := NovoGerenciador()
	posicoes := posicao.NovoRepositorio(pool)
	servico := NovoServico(repositorio, gerenciador, posicoes)
	c, err := servico.Criar(ctx, empresa, DTOCanal{Nome: "Área", TipoAcesso: "privado", LimiteParticipantes: 10})
	if err != nil {
		t.Fatal(err)
	}
	if err := posicoes.InserirLote(ctx, motorista, []posicao.Ponto{{Latitude: -28.46, Longitude: -52.20, CapturadoEm: time.Now()}}); err != nil {
		t.Fatal(err)
	}
	hub := gerenciador.ObterOuCriar(c.ID, 10)
	membro, _, _ := hub.Entrar(motorista, "Motorista")
	defer hub.Fechar()
	lat, lon, raio := -28.45, -52.20, 100
	_, err = servico.Editar(ctx, c.ID, empresa, DTOCanal{Nome: "Área", TipoAcesso: "privado", LimiteParticipantes: 10, GeocercaAtiva: true, CentroLatitude: &lat, CentroLongitude: &lon, RaioMetros: &raio})
	if err != nil {
		t.Fatal(err)
	}
	if hub.TemMembro(motorista) {
		t.Fatal("motorista fora da área continuou conectado")
	}
	evento := aguardarEvento(t, membro, "removido_canal")
	if evento.Dados["motivo"] != "geocerca" {
		t.Fatal("motivo de saída incorreto")
	}
}
func TestRaioECentroInvalidosSaoRecusados(t *testing.T) {
	for _, caso := range []struct {
		lat, lon float64
		raio     int
	}{{91, 0, 100}, {0, 181, 100}, {0, 0, 0}, {0, 0, -1}} {
		dto := DTOCanal{Nome: "Área", TipoAcesso: "privado", LimiteParticipantes: 10, GeocercaAtiva: true, CentroLatitude: &caso.lat, CentroLongitude: &caso.lon, RaioMetros: &caso.raio}
		if validarDTO(dto) == nil {
			t.Fatalf("área inválida aceita: %+v", caso)
		}
	}
}
