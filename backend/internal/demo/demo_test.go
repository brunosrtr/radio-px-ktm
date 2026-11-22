package demo

import (
	"math"
	"testing"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/geofence"
)

func TestRotaPassaPelosPorticosEPeloCentro(t *testing.T) {
	latC, lngC := -23.705, -46.895
	rota := Rota(latC, lngC, 66) // 65 trechos de 1 km entre -45 e +20 km
	if len(rota) != 66 {
		t.Fatalf("esperava 66 pontos, veio %d", len(rota))
	}
	primeiroLat, primeiroLng := Deslocar(latC, lngC, PorticosKm[0])
	if math.Abs(rota[0][0]-primeiroLat) > 1e-9 || math.Abs(rota[0][1]-primeiroLng) > 1e-9 {
		t.Fatal("a rota deveria começar no primeiro pórtico")
	}
	// O ponto 45 (km 0) é o centro.
	if d := geofence.Distancia(rota[45][0], rota[45][1], latC, lngC); d > 1 {
		t.Fatalf("a rota deveria passar pelo centro, ficou a %.0f m", d)
	}
	// Passos de cerca de 1 km.
	if d := geofence.Distancia(rota[0][0], rota[0][1], rota[1][0], rota[1][1]); d < 900 || d > 1100 {
		t.Fatalf("passo esperado ~1 km, veio %.0f m", d)
	}
}

func TestLerCentro(t *testing.T) {
	if _, _, err := LerCentro(CentroPadrao); err != nil {
		t.Fatal(err)
	}
	for _, invalido := range []string{"", "-23.5", "a,b", "-91,10", "10,181"} {
		if _, _, err := LerCentro(invalido); err == nil {
			t.Errorf("deveria recusar %q", invalido)
		}
	}
}
