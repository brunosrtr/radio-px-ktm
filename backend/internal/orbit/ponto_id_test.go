package orbit

import (
	"testing"
	"time"
)

func TestPontoIDRotativoEOpaco(t *testing.T) {
	manha := time.Date(2026, 11, 21, 9, 0, 0, 0, FusoContagem)
	noite := time.Date(2026, 11, 21, 23, 59, 0, 0, FusoContagem)
	amanha := time.Date(2026, 11, 22, 0, 1, 0, 0, FusoContagem)

	a := PontoID("segredo", "usuario-a", manha)
	if len(a) != 16 {
		t.Fatalf("ponto_id deveria ter 16 caracteres, tem %d", len(a))
	}
	if a != PontoID("segredo", "usuario-a", noite) {
		t.Fatal("no mesmo dia o ponto_id deveria ser o mesmo")
	}
	if a == PontoID("segredo", "usuario-a", amanha) {
		t.Fatal("no dia seguinte o ponto_id deveria mudar")
	}
	if a == PontoID("segredo", "usuario-b", manha) {
		t.Fatal("usuários diferentes não podem ter o mesmo ponto_id")
	}
	if a == PontoID("outro-segredo", "usuario-a", manha) {
		t.Fatal("o ponto_id deveria depender do segredo")
	}
	if a == "usuario-a" || len(a) == len("usuario-a") && a[:4] == "usua" {
		t.Fatal("o ponto_id não pode revelar o usuário")
	}
}

func TestPontosAtivosJanela(t *testing.T) {
	agora := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	p := NovoPontosAtivos().ComRelogio(func() time.Time { return agora })

	p.Registrar("a")
	agora = agora.Add(10 * time.Minute)
	p.Registrar("b")
	p.Registrar("b")
	if n := p.Contar(JanelaPontosAtivos); n != 2 {
		t.Fatalf("esperava 2 pontos ativos, veio %d", n)
	}
	agora = agora.Add(6 * time.Minute) // "a" visto há 16 min, "b" há 6 min
	if n := p.Contar(JanelaPontosAtivos); n != 1 {
		t.Fatalf("esperava 1 ponto ativo, veio %d", n)
	}
}
