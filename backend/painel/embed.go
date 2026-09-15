// Package painel embute os arquivos estáticos do painel web da empresa
// (mapa e consulta de trajeto), servidos pelo próprio backend Go — não há
// um "frontend" separado (plan.md).
package painel

import "embed"

//go:embed *.html
var Arquivos embed.FS
