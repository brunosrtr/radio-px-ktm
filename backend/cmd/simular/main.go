// Comando simular é o plano B da demonstração da Rede Orbit
// (roteiro-demo.md da 003-rede-orbit): sem depender do Bluetooth, envia pela
// API leituras de um caminhão "andando" pela rodovia simulada do seed, feitas
// por dois pontos de transmissão (as contas demo e admin_ktm). O painel
// mostra exatamente o que mostraria com os celulares de verdade.
//
//	go run ./cmd/simular -api http://localhost:8080 -orbit ORBIT-0A1B2D
//	docker compose exec backend /simular -orbit ORBIT-0A1B2D
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/demo"
)

func main() {
	api := flag.String("api", "http://localhost:8080", "endereço do backend")
	orbitID := flag.String("orbit", demo.Orbits[1].ID, "Orbit que vai \"andar\"")
	centro := flag.String("centro", demo.CentroPadrao, "o mesmo -centro usado no seed -demo")
	passos := flag.Int("passos", 30, "quantas leituras enviar ao longo da rodovia")
	intervalo := flag.Duration("intervalo", 4*time.Second, "tempo entre leituras")
	flag.Parse()

	latC, lngC, err := demo.LerCentro(*centro)
	if err != nil {
		log.Fatal(err)
	}
	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt)
	defer parar()

	cliente := &http.Client{Timeout: 10 * time.Second}
	tokens := make([]string, 0, len(demo.Contas))
	for _, c := range demo.Contas {
		token, err := entrar(cliente, *api, c.Email, c.Senha)
		if err != nil {
			log.Fatalf("não foi possível entrar com %s (rodou o seed -demo?): %v", c.Email, err)
		}
		tokens = append(tokens, token)
	}

	rota := demo.Rota(latC, lngC, *passos)
	fmt.Printf("simulando %s em %d leituras, uma a cada %s (Ctrl+C para parar)\n", *orbitID, len(rota), *intervalo)
	for i, p := range rota {
		token := tokens[i%len(tokens)]
		leitura := map[string]any{
			"orbit_id":        *orbitID,
			"latitude":        p[0],
			"longitude":       p[1],
			"precisao_metros": 5 + rand.Float64()*10,
			"rssi":            -60 - rand.IntN(20),
			"detectado_em":    time.Now().UTC().Format(time.RFC3339Nano),
		}
		if err := postar(cliente, *api+"/leituras", token, map[string]any{"leituras": []any{leitura}}); err != nil {
			log.Printf("leitura %d/%d falhou: %v", i+1, len(rota), err)
		} else {
			fmt.Printf("leitura %2d/%d  %.5f, %.5f  (ponto %d)\n", i+1, len(rota), p[0], p[1], i%len(tokens)+1)
		}
		select {
		case <-ctx.Done():
			fmt.Println("parado")
			return
		case <-time.After(*intervalo):
		}
	}
}

func entrar(cliente *http.Client, api, email, senha string) (string, error) {
	corpo, _ := json.Marshal(map[string]string{"identificador": email, "senha": senha})
	resp, err := cliente.Post(api+"/auth/login", "application/json", bytes.NewReader(corpo))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login respondeu %d", resp.StatusCode)
	}
	var sessao struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sessao); err != nil {
		return "", err
	}
	return sessao.Token, nil
}

func postar(cliente *http.Client, url, token string, dados any) error {
	corpo, _ := json.Marshal(dados)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(corpo))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := cliente.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("servidor respondeu %d", resp.StatusCode)
	}
	return nil
}
