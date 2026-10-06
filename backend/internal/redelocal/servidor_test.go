package redelocal

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIdentidadePersiste(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "identidade")
	a, err := Identidade(caminho)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Identidade(caminho)
	if err != nil || a != b || len(a) != 32 {
		t.Fatalf("identidade não persistiu: %q %q %v", a, b, err)
	}
	outro, err := Identidade(filepath.Join(t.TempDir(), "identidade"))
	if err != nil || outro == a {
		t.Fatal("instalações devem ter identidades diferentes")
	}
	if err := os.WriteFile(caminho, []byte("inválida"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Identidade(caminho); err == nil {
		t.Fatal("identidade inválida foi aceita")
	}
}

func TestInfoEQr(t *testing.T) {
	s := &Servidor{ID: strings.Repeat("a", 32), Nome: "Computador KTM", Porta: 8081}
	handler := s.Handler()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/info", nil))
	var dados map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &dados); err != nil {
		t.Fatal(err)
	}
	if dados["id"] != s.ID || dados["service"] != "radio-px-ktm" || dados["version"] != float64(1) {
		t.Fatal(dados)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("info não pode ficar em cache")
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/qr?endereco=http://evil.example", nil))
	if w.Code != 400 {
		t.Fatal("QR aceitou endereço externo")
	}
}
