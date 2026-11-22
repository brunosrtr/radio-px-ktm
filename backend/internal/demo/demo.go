// Package demo reúne os dados fixos da demonstração da Rede Orbit
// (roteiro-demo.md da 003-rede-orbit), usados pelo seed e pelo simulador de
// trajeto: contas de teste, Orbits virtuais e a "rodovia" simulada.
package demo

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/usuario"
)

// Conta é uma conta de demonstração (valores de teste, só para ambiente
// local).
type Conta struct {
	Nome, Apelido, Email, Senha, Papel string
}

// Contas de demonstração: o painel da KTM e o celular do Orbit virtual.
var Contas = []Conta{
	{"Administração KTM", "KTM", "admin.ktm@demo.radiopx.local", "demo-ktm-2026", usuario.PapelAdminKTM},
	{"Celular Orbit virtual", "Orbit demo", "demo@demo.radiopx.local", "demo-orbit-2026", usuario.PapelDemo},
}

// Orbits virtuais da demonstração.
var Orbits = []struct{ ID, Rotulo string }{
	{"ORBIT-0A1B2C", "Caminhão demo 1 · ABC1D23"},
	{"ORBIT-0A1B2D", "Caminhão demo 2 · KTM2A26"},
	{"ORBIT-0A1B2E", "Caminhão demo 3 · PXR4B07"},
}

// CentroPadrao é a BR-116 (Régis Bittencourt) perto de Itapecerica da Serra.
const CentroPadrao = "-23.705,-46.895"

// PorticosKm são as distâncias (km) do centro dos pórticos simulados ao
// longo de uma "rodovia" no sentido sudoeste. O centro fica de propósito
// entre dois pórticos: é onde a rede de celulares cobre o que os pontos
// fixos não cobrem.
var PorticosKm = []float64{-45, -30, -15, 20}

// Deslocar anda km quilômetros a partir do ponto no rumo sudoeste (km
// negativo = nordeste).
func Deslocar(lat, lng, km float64) (float64, float64) {
	d := km * math.Sqrt2 / 2
	return lat - d/111.0, lng - d/(111.0*math.Cos(lat*math.Pi/180))
}

// Rota devolve n pontos igualmente espaçados da rodovia simulada, do
// primeiro ao último pórtico.
func Rota(latC, lngC float64, n int) [][2]float64 {
	inicio, fim := PorticosKm[0], PorticosKm[len(PorticosKm)-1]
	pontos := make([][2]float64, 0, n)
	for i := range n {
		km := inicio + (fim-inicio)*float64(i)/float64(max(n-1, 1))
		lat, lng := Deslocar(latC, lngC, km)
		pontos = append(pontos, [2]float64{lat, lng})
	}
	return pontos
}

// LerCentro aceita "lat,lng".
func LerCentro(texto string) (float64, float64, error) {
	partes := strings.Split(texto, ",")
	if len(partes) != 2 {
		return 0, 0, fmt.Errorf("use lat,lng (ex.: %s)", CentroPadrao)
	}
	lat, err1 := strconv.ParseFloat(strings.TrimSpace(partes[0]), 64)
	lng, err2 := strconv.ParseFloat(strings.TrimSpace(partes[1]), 64)
	if err1 != nil || err2 != nil || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return 0, 0, fmt.Errorf("coordenadas inválidas: %q", texto)
	}
	return lat, lng, nil
}
