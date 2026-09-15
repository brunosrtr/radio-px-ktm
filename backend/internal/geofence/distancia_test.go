package geofence

import "testing"

func TestDistanciaEntreOMesmoPontoEZero(t *testing.T) {
	d := Distancia(-28.4497, -52.2003, -28.4497, -52.2003)
	if d != 0 {
		t.Fatalf("distância entre o mesmo ponto deveria ser 0, veio %f", d)
	}
}

func TestDistanciaEntrePontosConhecidos(t *testing.T) {
	// 0.001 grau de latitude equivale a ~111 metros em qualquer longitude.
	d := Distancia(0, 0, 0.001, 0)
	if d < 100 || d > 130 {
		t.Fatalf("esperava aproximadamente 111m, veio %f", d)
	}
}

func TestDentroDoRaioPontoDentro(t *testing.T) {
	centroLat, centroLon := -28.4497, -52.2003
	// ~55m de deslocamento em latitude.
	if !DentroDoRaio(centroLat+0.0005, centroLon, centroLat, centroLon, 100) {
		t.Fatal("ponto a ~55m deveria estar dentro do raio de 100m")
	}
}

func TestDentroDoRaioPontoFora(t *testing.T) {
	centroLat, centroLon := -28.4497, -52.2003
	// ~1113m de deslocamento em latitude.
	if DentroDoRaio(centroLat+0.01, centroLon, centroLat, centroLon, 100) {
		t.Fatal("ponto a ~1113m não deveria estar dentro do raio de 100m")
	}
}
