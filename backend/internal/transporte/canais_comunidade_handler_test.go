package transporte_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/config"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage/testutil"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/transporte"
)

func TestContratoCanaisDaComunidade(t *testing.T) {
	pool := testutil.AbrirPool(t)
	const segredo = "comunidade-teste"
	servidor := httptest.NewServer(transporte.NovoRoteador(config.Config{JWTSecret: segredo}, pool))
	defer servidor.Close()
	c := clienteTeste{t: t, url: servidor.URL}

	criadora := testutil.CriarUsuarioPublico(t, pool)
	convidado := testutil.CriarUsuarioPublico(t, pool)
	tokenCriadora, _ := auth.GerarToken(segredo, criadora, "", auth.PapelUsuario)
	tokenConvidado, _ := auth.GerarToken(segredo, convidado, "", auth.PapelUsuario)

	criado := c.chamar("POST", "/canais", tokenCriadora, map[string]any{
		"nome": "Comboio Sul", "visibilidade": "privado", "limite_participantes": 10,
		"geocerca": map[string]any{"latitude": -24.27, "longitude": -47.93, "raio_metros": 15000},
	}, http.StatusCreated)
	id, _ := criado["id"].(string)
	codigo, _ := criado["codigo_convite"].(string)
	if id == "" || len(codigo) != 9 || codigo[4] != '-' || criado["comunidade"] != true || criado["geocerca_ativa"] != true {
		t.Fatalf("criação fora do contrato: %v", criado)
	}

	c.chamar("GET", "/canais/"+id, tokenConvidado, nil, http.StatusForbidden)
	if r := c.chamar("POST", "/convites/ZZZZ-ZZZZ", tokenConvidado, nil, http.StatusNotFound); r["codigo"] != "convite_invalido" {
		t.Fatalf("esperava convite_invalido: %v", r)
	}
	entrou := c.chamar("POST", "/convites/"+codigo, tokenConvidado, nil, http.StatusOK)
	if entrou["id"] != id {
		t.Fatalf("convite deveria levar ao canal: %v", entrou)
	}

	detalhe := c.chamar("GET", "/canais/"+id, tokenConvidado, nil, http.StatusOK)
	if detalhe["codigo_convite"] != nil || detalhe["sou_criador"] != false || detalhe["pode_gerenciar"] != false {
		t.Fatalf("convidado não vê o código nem gerencia: %v", detalhe)
	}
	if d := c.chamar("GET", "/canais/"+id, tokenCriadora, nil, http.StatusOK); d["codigo_convite"] != codigo || d["pode_gerenciar"] != true {
		t.Fatalf("criadora vê o código: %v", d)
	}

	lista := c.chamar("GET", "/canais", tokenConvidado, nil, http.StatusOK)
	if len(lista["canais"].([]any)) != 0 {
		// O canal tem área e o convidado não tem posição: fica fora da lista
		// (FR-015), como os canais de empresa com geocerca.
		t.Fatalf("canal com área não deveria aparecer sem posição: %v", lista)
	}
	c.chamar("PUT", "/posicao-atual", tokenConvidado, map[string]any{"latitude": -24.27, "longitude": -47.93, "capturado_em": "2026-11-22T12:00:00Z"}, http.StatusNoContent)
	lista = c.chamar("GET", "/canais", tokenConvidado, nil, http.StatusOK)
	if itens := lista["canais"].([]any); len(itens) != 1 || itens[0].(map[string]any)["sou_criador"] != false {
		t.Fatalf("dentro da área o canal aparece: %v", lista)
	}

	c.chamar("PUT", "/canais/"+id+"/favorito", tokenConvidado, map[string]any{"favorito": true}, http.StatusNoContent)
	novo := c.chamar("POST", "/canais/"+id+"/convite", tokenCriadora, nil, http.StatusOK)
	if novo["codigo_convite"] == codigo {
		t.Fatal("novo convite deveria mudar o código")
	}
	c.chamar("POST", "/canais/"+id+"/convite", tokenConvidado, nil, http.StatusForbidden)
	if r := c.chamar("GET", "/canais/"+id+"/participantes", tokenCriadora, nil, http.StatusOK); r["participantes"] == nil {
		t.Fatalf("participantes fora do contrato: %v", r)
	}
	c.chamar("DELETE", "/canais/"+id+"/participantes/"+convidado, tokenConvidado, nil, http.StatusForbidden)
	c.chamar("DELETE", "/canais/"+id+"/participantes/"+convidado, tokenCriadora, nil, http.StatusNoContent)
	if r := c.chamar("POST", "/convites/"+novo["codigo_convite"].(string), tokenConvidado, nil, http.StatusForbidden); r["codigo"] != "banido" {
		t.Fatalf("removido não volta: %v", r)
	}

	publico := c.chamar("POST", "/canais", tokenCriadora, map[string]any{"nome": "Rodotrem Brasil " + criadora[:6], "visibilidade": "publico", "limite_participantes": 20}, http.StatusCreated)
	achados := c.chamar("GET", "/canais/explorar?q="+criadora[:6], tokenConvidado, nil, http.StatusOK)
	if itens := achados["canais"].([]any); len(itens) != 1 || itens[0].(map[string]any)["id"] != publico["id"] {
		t.Fatalf("explorar deveria achar só o público: %v", achados)
	}

	c.chamar("DELETE", "/canais/"+id, tokenConvidado, nil, http.StatusForbidden)
	c.chamar("DELETE", "/canais/"+id, tokenCriadora, nil, http.StatusNoContent)
	c.chamar("GET", "/canais/"+id, tokenCriadora, nil, http.StatusNotFound)
}
