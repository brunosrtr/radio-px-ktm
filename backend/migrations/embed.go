// Package migrations expõe os arquivos SQL versionados para o aplicador de
// migrações em internal/storage — mantém o SQL como fonte única de verdade
// (Princípio V da constituição), sem gerar código a partir dele.
package migrations

import "embed"

//go:embed *.sql
var Arquivos embed.FS
