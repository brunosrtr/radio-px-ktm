package transporte

import (
	"testing"
	"time"
)

func TestLimitadorJanelaDeslizante(t *testing.T) {
	agora := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	l := NovoLimitador(5, 10*time.Minute).ComRelogio(func() time.Time { return agora })

	for i := 1; i <= 5; i++ {
		if !l.Permitir("10.0.0.1") {
			t.Fatalf("tentativa %d deveria ser permitida", i)
		}
		agora = agora.Add(time.Minute)
	}
	if l.Permitir("10.0.0.1") {
		t.Fatal("a 6ª tentativa dentro da janela deveria ser recusada")
	}
	if !l.Permitir("10.0.0.2") {
		t.Fatal("outro IP não deveria ser afetado")
	}

	// A primeira tentativa foi às 12:00; às 12:10 + 1s ela sai da janela.
	agora = time.Date(2026, 11, 21, 12, 10, 1, 0, time.UTC)
	if !l.Permitir("10.0.0.1") {
		t.Fatal("depois que a tentativa mais antiga sai da janela, deveria voltar a permitir")
	}
	if l.Permitir("10.0.0.1") {
		t.Fatal("só uma vaga deveria ter sido liberada")
	}
}
