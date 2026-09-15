// Package testutil abre um pool de conexão com o Postgres real usado pelos
// testes de integração (Princípio IV da constituição: fila do hub, limite de
// participantes, geocerca e ingestão de posições são cobertos com banco de
// verdade, não mocks). Requer `docker compose up` rodando localmente.
package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
)

const databaseURLPadrao = "postgres://radiopx:changeme@localhost:5432/radiopx?sslmode=disable"

// AbrirPool conecta ao Postgres de teste (DATABASE_URL, ou o padrão do
// docker-compose local), aplica as migrações e devolve o pool, já registrado
// para fechar ao fim do teste. Se o Postgres não estiver acessível, o teste é
// pulado em vez de falhar — mantém `go test ./...` utilizável sem
// `docker compose up`.
func AbrirPool(t *testing.T) *storage.Pool {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = databaseURLPadrao
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pool, err := storage.Conectar(ctx, databaseURL)
	if err != nil {
		t.Skipf("Postgres de teste indisponível em %q (rode `docker compose up`): %v", databaseURL, err)
	}
	t.Cleanup(pool.Close)

	if err := pool.AplicarMigracoes(context.Background()); err != nil {
		t.Fatalf("erro ao aplicar migrações no banco de teste: %v", err)
	}

	return pool
}

// CriarEmpresa insere uma empresa de teste com CNPJ único e devolve seu ID.
func CriarEmpresa(t *testing.T, pool *storage.Pool) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		insert into empresa (razao_social, cnpj) values ($1, $2)
		returning id
	`, "Empresa Teste "+aleatorio(), aleatorio()[:14]).Scan(&id)
	if err != nil {
		t.Fatalf("erro ao criar empresa de teste: %v", err)
	}
	return id
}

// CriarUsuario insere um usuário de teste (login único) na empresa
// informada e devolve seu ID.
func CriarUsuario(t *testing.T, pool *storage.Pool, empresaID, papel string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		insert into usuario (empresa_id, nome, login, senha_hash, papel)
		values ($1, 'Usuário Teste', $2, 'hash', $3)
		returning id
	`, empresaID, "login-"+aleatorio(), papel).Scan(&id)
	if err != nil {
		t.Fatalf("erro ao criar usuário de teste: %v", err)
	}
	return id
}

func aleatorio() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
