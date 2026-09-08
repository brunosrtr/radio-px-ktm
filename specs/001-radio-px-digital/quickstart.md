# Quickstart

Feature: Rádio PX Digital v1 (`001-radio-px-digital`)

Este guia cobre o que a **Etapa 1 — Fundação** do plano de execução precisa
entregar (ver plan.md e o briefing original) e como validar seu critério de
aceite. As etapas seguintes (autenticação, canais, hub/WebSocket, áudio,
localização, painel, geocerca) são detalhadas em `tasks.md`, gerado por
`/speckit.tasks`.

## Pré-requisitos

- Docker e Docker Compose
- Go (versão compatível com a definida em `backend/go.mod`, a ser criado na
  Etapa 1)
- Flutter SDK (para o app, a partir da Etapa 2)

## Subindo o ambiente local

```bash
cp .env.example .env   # ajustar credenciais locais se necessário
docker compose up
```

O `docker-compose.yml` da Etapa 1 deve subir:

- `postgres`: banco local, com as migrações de `backend/migrations/`
  aplicadas automaticamente na inicialização (ou via um passo explícito de
  migração antes de o backend subir — a decisão de mecanismo de migração fica
  para a implementação, mas deve ser idempotente, conforme Princípio V da
  constituição).
- `backend`: API Go, expondo `/health`.

## Critério de aceite da Etapa 1

```bash
curl -f http://localhost:PORTA/health
```

Deve retornar sucesso (200) somente depois que:

1. `docker compose up` sobe o Postgres sem erro.
2. As migrações em `backend/migrations/` aplicam sem erro sobre um banco
   vazio (idempotência: rodar as migrações uma segunda vez sobre o mesmo
   banco não deve falhar nem duplicar estado).
3. O backend consegue se conectar ao Postgres e responder em `/health`.

Esse critério é o gate mínimo antes de iniciar a Etapa 2 (autenticação),
conforme o Princípio II da constituição (uma etapa por vez, com aceite
validado antes de seguir).

## Validando os contratos gerados nesta fase

- `contracts/rest-api.md` e `contracts/websocket-protocol.md` descrevem o
  comportamento esperado das rotas e do protocolo em tempo real — eles servem
  de referência para os testes de contrato que `/speckit.tasks` vai detalhar
  (particularmente os obrigatórios pelo Princípio IV: fila do hub, limite de
  participantes, geocerca e ingestão em lote de posições).
- `data-model.md` é a fonte de verdade do schema para a primeira migração —
  qualquer divergência entre o DDL do briefing original e o que for
  implementado deve ser corrigida aqui antes de codificar.

## Rodando os testes (a partir da Etapa 4 em diante)

```bash
# backend — testes unitários e de integração (Postgres real via docker compose)
cd backend && go test ./...

# app — a partir da Etapa 5
cd app && flutter test
```
