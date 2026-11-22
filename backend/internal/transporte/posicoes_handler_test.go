package transporte_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/config"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage/testutil"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/transporte"
)

// FR-034: contas do público não têm histórico de posições.
func TestPosicoesRecusaUsuarioSemEmpresa(t *testing.T) {
	pool := testutil.AbrirPool(t)
	const segredo = "posicoes-teste"
	publico := testutil.CriarUsuarioPublico(t, pool)
	token, _ := auth.GerarToken(segredo, publico, "", auth.PapelUsuario)

	servidor := httptest.NewServer(transporte.NovoRoteador(config.Config{JWTSecret: segredo}, pool))
	defer servidor.Close()

	corpo := []byte(`{"pontos":[{"latitude":-23.5,"longitude":-46.6,"capturado_em":"2026-11-21T12:00:00Z"}]}`)
	req, _ := http.NewRequest(http.MethodPost, servidor.URL+"/posicoes", bytes.NewReader(corpo))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status %d, esperado 403", resp.StatusCode)
	}

	var total int
	if err := pool.QueryRow(t.Context(), `select count(*) from posicao where usuario_id = $1`, publico).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 0 {
		t.Fatalf("nenhuma posição deveria ter sido gravada, encontrou %d", total)
	}

	// PUT /posicao-atual atualiza só a posição atual, sem histórico.
	put := func(corpo string, esperado int) {
		t.Helper()
		req, _ := http.NewRequest(http.MethodPut, servidor.URL+"/posicao-atual", bytes.NewReader([]byte(corpo)))
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != esperado {
			t.Fatalf("PUT /posicao-atual: status %d, esperado %d", resp.StatusCode, esperado)
		}
	}
	put(`{"latitude":-23.5,"longitude":-46.6,"capturado_em":"2026-11-21T12:00:00Z"}`, http.StatusNoContent)
	put(`{"latitude":-20.0,"longitude":-40.0,"capturado_em":"2026-11-21T11:00:00Z"}`, http.StatusNoContent)
	put(`{"latitude":-23.5,"longitude":-46.6}`, http.StatusBadRequest)

	var lat float64
	if err := pool.QueryRow(t.Context(), `select latitude::float8 from posicao_atual where usuario_id = $1`, publico).Scan(&lat); err != nil {
		t.Fatal(err)
	}
	if lat != -23.5 {
		t.Fatalf("um ponto mais antigo não deveria sobrescrever a posição atual: %v", lat)
	}
	if err := pool.QueryRow(t.Context(), `select count(*) from posicao where usuario_id = $1`, publico).Scan(&total); err != nil || total != 0 {
		t.Fatalf("PUT /posicao-atual não pode gravar histórico (%d, %v)", total, err)
	}
}
