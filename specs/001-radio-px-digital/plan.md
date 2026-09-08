# Implementation Plan: Rádio PX Digital — Canais de Voz, Localização e Painel da Empresa (v1)

**Branch**: `001-radio-px-digital` | **Date**: 2026-09-08 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-radio-px-digital/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Aplicativo Flutter (Android/iOS) para motoristas conversarem por voz push-to-talk
em canais geridos por suas empresas, com fila de reprodução em memória (sem
qualquer persistência de áudio), controle de acesso por empresa e por geocerca,
mais um backend Go que também coleta e retém a localização dos motoristas em
segundo plano (com fila offline no dispositivo) e expõe um painel web para a
empresa acompanhar posição atual e trajeto. O core técnico é um hub de canais
em memória no backend Go (uma goroutine por canal, fila bufferizada nativa da
linguagem) conectado via WebSocket; REST cobre autenticação, CRUD de canais e
ingestão de lotes de posição; PostgreSQL guarda apenas dados estruturados
(nunca áudio), com SQL escrito à mão.

## Technical Context

**Language/Version**: Go (backend + painel), Dart/Flutter (app móvel)
**Primary Dependencies**: Backend — `pgx` (Postgres), `chi` (roteamento HTTP), `coder/websocket` (WebSocket), `golang-jwt` (JWT). App — `riverpod` (estado), `dio` (HTTP), `web_socket_channel`, `geolocator` (localização), `flutter_foreground_task` (segundo plano Android), `flutter_sound` (gravação/reprodução Opus em streaming). Painel — templates HTML nativos do Go + Leaflet (mapa).
**Storage**: PostgreSQL (dados estruturados: empresas, usuários, canais, participações, posições — nunca áudio). Fila offline de posições no dispositivo: Hive (ver research.md para a decisão entre Hive e Drift).
**Testing**: Backend — `go test` (testes unitários e de integração com Postgres real via `docker compose`, cobrindo obrigatoriamente fila do hub, limite de participantes, cálculo de geocerca e ingestão de lote de posições — Princípio IV da constituição). App — `flutter_test`.
**Target Platform**: Android e iOS (app); Linux server em container (backend + painel), acessado via navegador desktop/mobile para o painel.
**Project Type**: Aplicação móvel + API/backend com painel web administrativo embutido (não é um "frontend" separado — o painel é servido pelo próprio backend Go).
**Performance Goals**: Latência de fala-para-escuta perceptivelmente baixa (meta: <2s em rede móvel normal, RNF07/SC-001); suportar 20 participantes simultâneos por canal (RNF02/SC-002); ingestão de posição em lote, não ponto a ponto (RNF15).
**Constraints**: Áudio nunca persiste em nenhuma camada (Princípio I, RF09/RNF05/RNF06); mensagem de voz limitada a 90s, validada no servidor (RF04/RNF04); fila de canal limitada a 10 mensagens (RF06/RNF03); áudio comprimido a no máximo 24 kbps (RNF14); app funcional com GPS em segundo plano e sem sinal, sincronizando em lote ao reconectar (RF02, RF31, RNF09, RNF15); backend de instância única na v1, sem estado de hub compartilhado entre processos (ver Princípio VI).
**Scale/Scope**: Múltiplas empresas (multi-tenant lógico via `empresa_id`), até 20 motoristas simultâneos por canal, retenção de histórico de posição por 90 dias (RNF16); volume estimado de ~2.900 linhas de posição por motorista/dia se em movimento contínuo (ver notas do modelo de dados).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Referência: [constitution.md](/.specify/memory/constitution.md) v1.0.0.

| Princípio | Gate | Status |
|---|---|---|
| I. Áudio nunca é persistido | O hub de canais mantém fila e buffers de transmissão apenas em memória do processo Go (`chan *Transmissao`, `chunks [][]byte`); nenhuma tabela do modelo de dados (seção 6 / data-model.md) armazena payload de áudio; o buffer é zerado após a distribuição aos ouvintes. | PASS |
| II. Uma etapa por vez, com aceite validado | Este plano não decide sozinho a ordem de execução — ela é a das 8 etapas já definidas no briefing (fundação → auth → canais → hub/WS → áudio no app → localização → painel → geocerca/segundo plano), a ser detalhada em tasks.md por `/speckit.tasks`. | PASS (a ser operacionalizado em tasks.md) |
| III. Nada de requisito inventado | Nenhum `[NEEDS CLARIFICATION]` restou na spec; as 4 pendências reais do briefing foram preservadas em "Pendências para validar com a KTM" no spec.md, com um valor padrão assumido para cada uma até a validação. | PASS |
| IV. Teste obrigatório nas regras críticas | Contracts e research.md preveem testes automatizados obrigatórios para: fila FIFO/limite de 10 mensagens do hub, limite de participantes por canal, cálculo/verificação de geocerca, e ingestão em lote de posições com atualização condicional de `posicao_atual`. | PASS (gate reforçado em tasks.md) |
| V. Dados explícitos, sem ORM | Backend usa `pgx` com SQL escrito à mão (ver data-model.md e migrations/); nenhum ORM Go é introduzido. O uso de Hive no app (armazenamento local de fila offline) não é alcançado por este princípio, que se aplica ao acesso ao PostgreSQL. | PASS |
| VI. Simplicidade adequada ao escopo acadêmico | Hub de canais local a um único processo Go (sem Redis/Pub-Sub); interface do hub isolada para permitir evolução futura sem reescrever handlers (ver research.md). | PASS |

Nenhuma violação identificada — a seção "Complexity Tracking" abaixo permanece vazia.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
backend/
├── cmd/api/main.go
├── internal/
│   ├── config/       # carregamento de variáveis de ambiente
│   ├── auth/         # JWT, hash de senha, middleware de autenticação
│   ├── empresa/      # serviço e repositório
│   ├── usuario/      # serviço e repositório
│   ├── canal/        # serviço, repositório, hub, fila
│   ├── posicao/      # serviço, repositório, ingestão em lote
│   ├── geofence/      # cálculo de distância e verificação periódica
│   ├── ws/           # upgrade, roteamento de eventos, conexões
│   ├── transporte/   # handlers HTTP, rotas, DTOs
│   └── storage/      # pool pgx, migrações
├── painel/           # templates HTML e assets do painel da empresa (Leaflet)
├── migrations/       # SQL versionado e idempotente
└── (testes Go colocalizados por pacote: *_test.go, mais testes de integração
    em internal/canal e internal/posicao usando Postgres real via docker compose)

app/
├── lib/
│   ├── main.dart
│   ├── core/            # tema, rotas, constantes, cliente HTTP
│   ├── models/          # entidades compartilhadas
│   ├── data/            # repositórios e fonte de dados local (fila offline)
│   ├── services/
│   │   ├── audio_service.dart        # gravação/reprodução Opus em streaming
│   │   ├── canal_service.dart        # conexão WebSocket e eventos
│   │   ├── localizacao_service.dart  # coleta, fila offline, envio em lote
│   │   └── background_service.dart   # foreground service e ciclo de vida
│   └── features/
│       ├── auth/
│       ├── canais/
│       └── configuracoes/
└── test/                # flutter_test: unit + widget

docker-compose.yml        # Postgres + backend, para a Etapa 1
.env.example
```

**Structure Decision**: Estrutura de dois componentes de primeira classe no
mesmo repositório — `backend/` (Go: API REST, hub de WebSocket e painel web
administrativo, todos servidos pelo mesmo processo) e `app/` (Flutter, único
código-fonte para Android e iOS). Não há um "frontend" web separado: o painel
da empresa é HTML renderizado pelo próprio backend Go, por isso não se usa o
layout padrão de "backend/ + frontend/" — o painel vive dentro de
`backend/painel/`. Layout e nomes de pastas replicam exatamente as seções 7 e
10 do briefing original da KTM, para manter coerência com a defesa do TCC.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
