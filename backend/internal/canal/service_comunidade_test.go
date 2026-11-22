package canal

import (
	"context"
	"errors"
	"testing"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/posicao"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage/testutil"
)

func TestCanaisDaComunidade(t *testing.T) {
	pool := testutil.AbrirPool(t)
	ctx := context.Background()
	gerenciador := NovoGerenciador()
	servico := NovoServico(NovoRepositorio(pool), gerenciador, posicao.NovoRepositorio(pool))

	ana := Autor{UsuarioID: testutil.CriarUsuarioPublico(t, pool)}
	beto := Autor{UsuarioID: testutil.CriarUsuarioPublico(t, pool)}
	caio := Autor{UsuarioID: testutil.CriarUsuarioPublico(t, pool)}
	contem := func(lista []CanalListado, id string) bool {
		for _, c := range lista {
			if c.ID == id {
				return true
			}
		}
		return false
	}
	listar := func(a Autor) []CanalListado {
		t.Helper()
		l, err := servico.ListarParaMotorista(ctx, "", a.UsuarioID)
		if err != nil {
			t.Fatal(err)
		}
		return l
	}

	privado, err := servico.CriarComo(ctx, ana, DTOCanal{Nome: "Comboio Sul " + ana.UsuarioID[:6], Visibilidade: "privado", LimiteParticipantes: 10})
	if err != nil {
		t.Fatal(err)
	}
	if privado.EmpresaID != "" || privado.CriadorID != ana.UsuarioID || NormalizarCodigoConvite(privado.CodigoConvite) == "" {
		t.Fatalf("canal da comunidade criado errado: %+v", privado)
	}
	publico, err := servico.CriarComo(ctx, ana, DTOCanal{Nome: "Serra do Cafezal " + ana.UsuarioID[:6], Visibilidade: "publico", LimiteParticipantes: 20})
	if err != nil {
		t.Fatal(err)
	}
	if publico.CodigoConvite != "" {
		t.Fatal("canal público não deve ter convite")
	}

	t.Run("visibilidade inválida é recusada", func(t *testing.T) {
		if _, err := servico.CriarComo(ctx, ana, DTOCanal{Nome: "x", Visibilidade: "secreto", LimiteParticipantes: 10}); !errors.Is(err, ErrCanalInvalido) {
			t.Fatalf("esperava ErrCanalInvalido, veio %v", err)
		}
	})

	t.Run("privado não aparece na busca; público aparece", func(t *testing.T) {
		achados, err := servico.Explorar(ctx, beto.UsuarioID, ana.UsuarioID[:6], nil, nil, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(achados) != 1 || achados[0].ID != publico.ID {
			t.Fatalf("busca deveria achar só o público: %+v", achados)
		}
		if _, err := servico.ObterParaEntrada(ctx, privado.ID, "", beto.UsuarioID); !errors.Is(err, ErrSemPermissao) {
			t.Fatalf("sem convite não entra: %v", err)
		}
		if _, err := servico.ObterParaEntrada(ctx, publico.ID, "", beto.UsuarioID); err != nil {
			t.Fatalf("público deveria permitir entrada: %v", err)
		}
	})

	t.Run("convite libera o privado e ele passa a ser listado", func(t *testing.T) {
		if contem(listar(beto), privado.ID) {
			t.Fatal("antes do convite o canal não deveria ser listado")
		}
		if _, err := servico.EntrarPorConvite(ctx, FormatarCodigoConvite(privado.CodigoConvite), beto.UsuarioID); err != nil {
			t.Fatal(err)
		}
		if _, err := servico.ObterParaEntrada(ctx, privado.ID, "", beto.UsuarioID); err != nil {
			t.Fatalf("com convite deveria entrar: %v", err)
		}
		if !contem(listar(beto), privado.ID) {
			t.Fatal("depois do convite o canal deveria ser listado")
		}
		if !contem(listar(ana), privado.ID) || !contem(listar(ana), publico.ID) {
			t.Fatal("a criadora deveria ver os próprios canais")
		}
	})

	t.Run("novo convite invalida o antigo, mas quem entrou continua", func(t *testing.T) {
		antigo := privado.CodigoConvite
		novo, err := servico.NovoConvite(ctx, privado.ID, ana)
		if err != nil {
			t.Fatal(err)
		}
		if novo == antigo {
			t.Fatal("o código deveria mudar")
		}
		if _, err := servico.EntrarPorConvite(ctx, antigo, caio.UsuarioID); !errors.Is(err, ErrNaoEncontrado) {
			t.Fatalf("convite antigo deveria falhar: %v", err)
		}
		if _, err := servico.ObterParaEntrada(ctx, privado.ID, "", beto.UsuarioID); err != nil {
			t.Fatalf("quem já entrou continua: %v", err)
		}
		if _, err := servico.NovoConvite(ctx, privado.ID, beto); !errors.Is(err, ErrSemPermissao) {
			t.Fatalf("só a criadora gera convite: %v", err)
		}
		privado.CodigoConvite = novo
	})

	t.Run("favorito traz o público para a lista", func(t *testing.T) {
		if err := servico.Favoritar(ctx, publico.ID, caio, true); err != nil {
			t.Fatal(err)
		}
		if !contem(listar(caio), publico.ID) {
			t.Fatal("favorito deveria aparecer na lista")
		}
		if err := servico.Favoritar(ctx, privado.ID, caio, true); !errors.Is(err, ErrSemPermissao) {
			t.Fatalf("não dá para favoritar privado sem convite: %v", err)
		}
	})

	t.Run("removido sai na hora e não volta nem por convite", func(t *testing.T) {
		hub := gerenciador.ObterOuCriar(privado.ID, 10)
		membro, _, err := hub.Entrar(beto.UsuarioID, "Beto")
		if err != nil {
			t.Fatal(err)
		}
		if err := servico.Remover(ctx, privado.ID, beto, ana.UsuarioID); !errors.Is(err, ErrSemPermissao) {
			t.Fatalf("participante não remove ninguém: %v", err)
		}
		if err := servico.Remover(ctx, privado.ID, ana, ana.UsuarioID); !errors.Is(err, ErrNaoPodeRemoverSiMesmo) {
			t.Fatalf("criadora não se remove: %v", err)
		}
		if err := servico.Remover(ctx, privado.ID, ana, beto.UsuarioID); err != nil {
			t.Fatal(err)
		}
		if e := aguardarEvento(t, membro, "removido_canal"); e.Dados["motivo"] != "removido" {
			t.Fatalf("motivo errado: %+v", e)
		}
		if _, err := servico.ObterParaEntrada(ctx, privado.ID, "", beto.UsuarioID); !errors.Is(err, ErrBanido) {
			t.Fatalf("removido não volta: %v", err)
		}
		if _, err := servico.EntrarPorConvite(ctx, privado.CodigoConvite, beto.UsuarioID); !errors.Is(err, ErrBanido) {
			t.Fatalf("removido não volta por convite: %v", err)
		}
		if contem(listar(beto), privado.ID) {
			t.Fatal("removido não deveria ver o canal")
		}
	})

	t.Run("editar e apagar só pela criadora; apagar desconecta", func(t *testing.T) {
		if _, err := servico.EditarComo(ctx, publico.ID, caio, DTOCanal{Nome: "x", LimiteParticipantes: 5}); !errors.Is(err, ErrSemPermissao) {
			t.Fatalf("só a criadora edita: %v", err)
		}
		editado, err := servico.EditarComo(ctx, publico.ID, ana, DTOCanal{Nome: publico.Nome, Visibilidade: "privado", LimiteParticipantes: 5})
		if err != nil {
			t.Fatal(err)
		}
		if editado.Visibilidade != "privado" || NormalizarCodigoConvite(editado.CodigoConvite) == "" {
			t.Fatalf("ao virar privado deveria ganhar convite: %+v", editado)
		}

		hub := gerenciador.ObterOuCriar(publico.ID, 5)
		membro, _, err := hub.Entrar(caio.UsuarioID, "Caio")
		if err != nil {
			t.Fatal(err)
		}
		if err := servico.Apagar(ctx, publico.ID, caio); !errors.Is(err, ErrSemPermissao) {
			t.Fatalf("só a criadora apaga: %v", err)
		}
		if err := servico.Apagar(ctx, publico.ID, ana); err != nil {
			t.Fatal(err)
		}
		if e := aguardarEvento(t, membro, "removido_canal"); e.Dados["motivo"] != "canal_apagado" {
			t.Fatalf("motivo errado: %+v", e)
		}
		if _, err := servico.ObterParaEntrada(ctx, publico.ID, "", ana.UsuarioID); !errors.Is(err, ErrNaoEncontrado) {
			t.Fatalf("canal apagado não existe mais: %v", err)
		}
	})
}

func TestExplorarPertoDeMim(t *testing.T) {
	pool := testutil.AbrirPool(t)
	ctx := context.Background()
	servico := NovoServico(NovoRepositorio(pool), NovoGerenciador(), posicao.NovoRepositorio(pool))
	ana := Autor{UsuarioID: testutil.CriarUsuarioPublico(t, pool)}
	sufixo := ana.UsuarioID[:6]

	area := func(lat, lng float64, raio int) DTOCanal {
		return DTOCanal{Visibilidade: "publico", LimiteParticipantes: 10, GeocercaAtiva: true, CentroLatitude: &lat, CentroLongitude: &lng, RaioMetros: &raio}
	}
	perto := area(-23.70, -46.90, 5000)
	perto.Nome = "Perto " + sufixo
	longe := area(-15.79, -47.88, 5000) // Brasília
	longe.Nome = "Longe " + sufixo
	semArea := DTOCanal{Nome: "Nacional " + sufixo, Visibilidade: "publico", LimiteParticipantes: 10}
	for _, d := range []DTOCanal{perto, longe, semArea} {
		if _, err := servico.CriarComo(ctx, ana, d); err != nil {
			t.Fatal(err)
		}
	}

	lat, lng := -23.55, -46.63 // São Paulo, ~30 km do centro de "Perto"
	achados, err := servico.Explorar(ctx, ana.UsuarioID, sufixo, &lat, &lng, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(achados) != 1 || achados[0].Nome != perto.Nome || *achados[0].DistanciaMetros <= 0 || *achados[0].DistanciaMetros > DistanciaPerto {
		t.Fatalf("perto de mim deveria trazer só o canal próximo: %+v", achados)
	}

	todos, err := servico.Explorar(ctx, ana.UsuarioID, sufixo, &lat, &lng, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 3 {
		t.Fatalf("todos os públicos deveriam vir: %d", len(todos))
	}
}
