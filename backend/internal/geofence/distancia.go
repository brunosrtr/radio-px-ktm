// Package geofence calcula distância geográfica e decide entrada/remoção de
// canais por raio (research.md §6). Não depende do pacote canal — quem
// precisa acionar uma remoção implementa a interface Removedor definida em
// verificador.go, evitando um ciclo de importação entre os dois pacotes.
package geofence

import "math"

const raioTerraMetros = 6371000.0

// Distancia calcula, em metros, a distância entre dois pontos usando a
// fórmula de Haversine.
func Distancia(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return raioTerraMetros * c
}

// DentroDoRaio indica se o ponto (lat, lon) está a até raioMetros do centro
// informado (FR-014/FR-015).
func DentroDoRaio(lat, lon, centroLat, centroLon float64, raioMetros int) bool {
	return Distancia(lat, lon, centroLat, centroLon) <= float64(raioMetros)
}
