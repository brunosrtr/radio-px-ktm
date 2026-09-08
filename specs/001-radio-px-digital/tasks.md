# Tasks: Rádio PX Digital — Canais de Voz, Localização e Painel da Empresa (v1)

**Input**: Design documents from `/specs/001-radio-px-digital/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Incluídos e obrigatórios para as quatro regras que a constituição do
projeto (Princípio IV) exige testar: comportamento da fila do hub, limite de
participantes, cálculo/verificação de geocerca e ingestão em lote de
posições. Demais testes (contrato de auth, listagem filtrada de canais, etc.)
também foram incluídos por cobrirem regras de autorização multiempresa
sensíveis (RNF11/RNF12).

**Organization**: Tarefas agrupadas por user story (US1–US5, na ordem de
prioridade do spec.md) para permitir implementação e teste independentes de
cada uma.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Pode rodar em paralelo (arquivos diferentes, sem dependência de tarefa incompleta)
- **[Story]**: A qual user story a tarefa pertence (US1–US5)
- Caminhos de arquivo exatos em cada descrição

## Path Conventions

Conforme plan.md: `backend/` (Go — API, hub WebSocket, painel web) e `app/`
(Flutter — Android/iOS). Sem diretório `frontend/` separado — o painel vive em
`backend/painel/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Inicialização do projeto — Etapa 1 do briefing ("Fundação")

- [ ] T001 Criar estrutura de diretórios do backend (`backend/cmd/api/`, `backend/internal/{config,auth,empresa,usuario,canal,posicao,geofence,ws,transporte,storage}/`, `backend/painel/`, `backend/migrations/`) e do app (`app/lib/{core,models,data,services,features}/`, `app/test/`) conforme plan.md
- [ ] T002 Inicializar módulo Go em `backend/go.mod` com as dependências `pgx`, `chi`, `coder/websocket`, `golang-jwt`
- [ ] T003 [P] Inicializar projeto Flutter em `app/pubspec.yaml` com `riverpod`, `dio`, `web_socket_channel`, `geolocator`, `flutter_foreground_task`, `flutter_sound`, `hive`
- [ ] T004 [P] Criar `docker-compose.yml` na raiz do repositório com os serviços `postgres` e `backend`
- [ ] T005 [P] Criar `.env.example` na raiz com as variáveis de ambiente iniciais (conexão Postgres, segredo JWT, porta do backend)
- [ ] T006 [P] Configurar lint/format do backend (`gofmt` + `golangci-lint`) e do app (`flutter analyze`)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Infraestrutura que TODA user story depende

**⚠️ CRITICAL**: Nenhuma user story começa antes desta fase estar completa

- [ ] T007 Criar migração inicial `backend/migrations/0001_init.sql` com o DDL completo de data-model.md (`empresa`, `usuario`, `dispositivo`, `canal`, `canal_empresa`, `participacao_canal`, `preferencia_canal`, `posicao`, `posicao_atual`), idempotente
- [ ] T008 [P] Implementar carregamento de configuração por variáveis de ambiente em `backend/internal/config/config.go`
- [ ] T009 Implementar pool de conexão `pgx` e aplicação idempotente das migrações em `backend/internal/storage/storage.go` (depende de T007, T008)
- [ ] T010 [P] Implementar hash de senha (bcrypt) e geração/validação de JWT com claims `usuario_id`, `empresa_id`, `papel` em `backend/internal/auth/auth.go`
- [ ] T011 [P] Implementar middleware HTTP e de upgrade WebSocket que valida o JWT e injeta `usuario_id`/`empresa_id`/`papel` no contexto da requisição, retornando 401 quando ausente/inválido (RNF11) em `backend/internal/auth/middleware.go` (depende de T010)
- [ ] T012 Configurar roteador base (`chi`) e o endpoint `GET /health` em `backend/internal/transporte/router.go` e `backend/cmd/api/main.go` (depende de T009)
- [ ] T013 [P] Criar cliente HTTP (`dio`) com interceptor de autenticação (anexa o Bearer token) em `app/lib/core/http_client.dart`
- [ ] T014 [P] Criar rotas e tema base do app em `app/lib/core/`

**Checkpoint**: `docker compose up` sobe Postgres + backend, migrações aplicam sem erro, `GET /health` responde — critério de aceite da Etapa 1 do briefing (ver quickstart.md).

---

## Phase 3: User Story 1 - Comunicação por voz em canais (Priority: P1) 🎯 MVP

**Goal**: Um motorista fala num canal e outro motorista conectado ao mesmo canal ouve, respeitando fila FIFO de até 10 mensagens, limite de 90s por mensagem, e descarte definitivo do áudio após a reprodução.

**Independent Test**: Com dois usuários de teste inseridos diretamente no mesmo canal (via seed/fixture, sem telas de gestão ainda existirem), um motorista grava e envia uma mensagem de voz; o outro a recebe e ouve na íntegra e na ordem correta, e ela deixa de existir em qualquer lugar do sistema assim que termina de tocar.

### Tests for User Story 1 ⚠️

> Escrever estes testes PRIMEIRO; devem falhar antes da implementação.

- [ ] T015 [P] [US1] Teste unitário: fila do hub aceita até 10 mensagens e recusa a 11ª com `fila_cheia`, em `backend/internal/canal/hub_test.go`
- [ ] T016 [P] [US1] Teste unitário: fila reproduz mensagens estritamente na ordem de chegada (FIFO) e zera o buffer de chunks após cada reprodução, em `backend/internal/canal/hub_test.go`
- [ ] T017 [P] [US1] Teste unitário: transmissão é cortada automaticamente ao atingir 90 segundos, em `backend/internal/canal/hub_test.go`
- [ ] T018 [P] [US1] Teste de integração do protocolo WebSocket: `entrar_canal` → `solicitar_slot` → `slot_concedido` → frames binários → `inicio_reproducao`/`fim_reproducao`, em `backend/internal/ws/handler_test.go`

### Implementation for User Story 1

- [ ] T019 [P] [US1] Criar tipos `Canal` e `Transmissao` (fila em memória `chan *Transmissao` capacidade 10, buffer de chunks `[][]byte`) em `backend/internal/canal/hub.go`
- [ ] T020 [US1] Implementar o hub por canal: goroutine consumidora, distribuição FIFO aos membros não silenciados, buffer zerado após distribuir, em `backend/internal/canal/hub.go` (depende de T019)
- [ ] T021 [US1] Implementar corte automático de transmissão ao atingir 90 segundos dentro do hub em `backend/internal/canal/hub.go` (depende de T020)
- [ ] T022 [US1] Implementar registro básico de `participacao_canal` (entrar/sair, sem checagem de limite ou geocerca ainda — essas vêm nas fases US2/US3) em `backend/internal/canal/repositorio.go`
- [ ] T023 [US1] Implementar handler de upgrade WebSocket e roteamento dos eventos `entrar_canal`, `solicitar_slot`, `finalizar_transmissao`, `ping` em `backend/internal/ws/handler.go` (depende de T011, T020, T022)
- [ ] T024 [US1] Implementar recepção/encaminhamento de frames binários de áudio associados à transmissão ativa da conexão, emitindo `inicio_reproducao` (com nome do remetente) e `fim_reproducao`, em `backend/internal/ws/handler.go` (depende de T023)
- [ ] T025 [US1] Conectar a rota `GET /ws` ao roteador em `backend/internal/transporte/router.go` (depende de T012, T023)
- [ ] T026 [P] [US1] Implementar `audio_service.dart` (gravação/reprodução em streaming, codec Opus, ≤24 kbps, sem gravar em arquivo local) em `app/lib/services/audio_service.dart`
- [ ] T027 [P] [US1] Implementar `canal_service.dart` (conexão WebSocket, eventos de controle, envio/recepção de frames binários) em `app/lib/services/canal_service.dart`
- [ ] T028 [US1] Implementar a tela do canal ativo (botão push-to-talk, bloqueio quando a fila está cheia, exibição do nome do remetente durante a reprodução) em `app/lib/features/canais/canal_ativo_page.dart` (depende de T026, T027)

**Checkpoint**: User Story 1 completa e testável de forma independente.

---

## Phase 4: User Story 2 - Gestão de canais e controle de acesso pela empresa (Priority: P1)

**Goal**: Um administrador cria/gerencia canais da sua empresa (nome, descrição, limite de participantes, compartilhamento com parceiras); motoristas só veem os canais autorizados para sua empresa.

**Independent Test**: Um administrador cria um canal e define seu limite de participantes; motoristas da própria empresa passam a vê-lo na lista, motoristas de empresa não autorizada não o veem — sem depender de voz ou geolocalização estarem funcionando.

### Tests for User Story 2 ⚠️

- [ ] T029 [P] [US2] Teste de integração: criação/edição de canal valida `limite_participantes` em `{5,10,15,20}` e a constraint de geocerca completa, em `backend/internal/canal/service_test.go`
- [ ] T030 [P] [US2] Teste de integração: `GET /canais` só retorna canal privado da própria empresa e canais compartilhados liberados para ela, em `backend/internal/canal/service_test.go`
- [ ] T031 [P] [US2] Teste de integração: entrada em canal é recusada ao atingir o limite de participantes, em `backend/internal/canal/hub_test.go`

### Implementation for User Story 2

- [ ] T032 [P] [US2] Criar entidade e repositório de `Empresa` em `backend/internal/empresa/`
- [ ] T033 [P] [US2] Criar entidade e repositório de `Usuario` em `backend/internal/usuario/`
- [ ] T034 [US2] Implementar repositório de `Canal` (CRUD) em `backend/internal/canal/repositorio.go` (depende de T032)
- [ ] T035 [US2] Implementar repositório de `canal_empresa` (liberar/revogar acesso de empresa parceira) em `backend/internal/canal/repositorio.go` (depende de T034)
- [ ] T036 [US2] Implementar repositório de `preferencia_canal` (silenciar/reativar) em `backend/internal/canal/repositorio.go`
- [ ] T037 [US2] Implementar serviço de canal: criar/editar, liberar/revogar empresa parceira, listar canais filtrados por autorização de empresa (RNF12) em `backend/internal/canal/service.go` (depende de T034, T035)
- [ ] T038 [US2] Adicionar checagem de limite de participantes ao `entrar_canal` no hub (recusa quando `count(saiu_em is null) >= limite_participantes`) em `backend/internal/canal/hub.go` (depende de T020, T022, T037)
- [ ] T039 [US2] Implementar saída manual (`sair_canal`, grava `motivo_saida='manual'`) e troca de canal no handler WebSocket em `backend/internal/ws/handler.go` (depende de T023)
- [ ] T040 [US2] Implementar handlers `POST/PATCH /canais`, `POST/DELETE /canais/{id}/empresas`, `PUT /canais/{id}/preferencia`, `GET /canais` em `backend/internal/transporte/canais_handler.go` (depende de T037)
- [ ] T041 [US2] Implementar handlers `POST /auth/login` e `GET /me` em `backend/internal/transporte/auth_handler.go` (depende de T010, T033)
- [ ] T042 [US2] Criar seed com uma empresa e três usuários de teste em `backend/cmd/seed/main.go`
- [ ] T043 [P] [US2] Implementar tela de login em `app/lib/features/auth/login_page.dart` (depende de T013)
- [ ] T044 [P] [US2] Implementar tela de lista de canais com opções de silenciar, sair e alternar em `app/lib/features/canais/lista_canais_page.dart`
- [ ] T045 [US2] Conectar login → armazenamento do token → navegação para a lista de canais em `app/lib/features/auth/` (depende de T043)

**Checkpoint**: User Stories 1 e 2 funcionam de forma independente e integrada.

---

## Phase 5: User Story 3 - Restrição de canais por geolocalização (geocerca) (Priority: P2)

**Goal**: Canais com geocerca só aceitam motoristas dentro do raio configurado, e removem automaticamente (com aviso) quem sai da área.

**Independent Test**: Com um canal configurado com geocerca, um motorista simulado dentro do raio entra normalmente; ao simular deslocamento para fora do raio, é removido automaticamente e avisado, sem depender de nenhuma outra funcionalidade.

### Tests for User Story 3 ⚠️

- [ ] T046 [P] [US3] Teste unitário da função de distância Haversine (pontos dentro e fora do raio) em `backend/internal/geofence/distancia_test.go`
- [ ] T047 [P] [US3] Teste de integração: entrada recusada fora do raio (`fora_da_area`) e remoção automática com `motivo_saida='geocerca'` ao sair do raio, em `backend/internal/geofence/geofence_test.go`

### Implementation for User Story 3

- [ ] T048 [P] [US3] Implementar função de distância Haversine em `backend/internal/geofence/distancia.go`
- [ ] T049 [US3] Adicionar checagem de geocerca ao `entrar_canal`, usando a última `posicao_atual` do motorista, em `backend/internal/canal/hub.go` (depende de T038, T048)
- [ ] T050 [US3] Implementar verificador de geocerca acionado a cada atualização de posição, que remove motoristas fora do raio, envia `removido_canal` e grava `motivo_saida='geocerca'`, em `backend/internal/geofence/verificador.go` (depende de T048, T020)
- [ ] T051 [US3] Filtrar `GET /canais` pela geocerca usando a `posicao_atual` do motorista em `backend/internal/canal/service.go` (depende de T037, T048)
- [ ] T052 [P] [US3] Exibir aviso no app quando `removido_canal` chega com motivo geocerca, em `app/lib/features/canais/canal_ativo_page.dart` (depende de T028)

**Checkpoint**: User Stories 1, 2 e 3 funcionam de forma independente e integrada.

---

## Phase 6: User Story 4 - Rastreamento contínuo de localização (Priority: P2)

**Goal**: O app registra a posição do motorista periodicamente, mesmo em segundo plano e sem sinal, sincronizando em lote ao reconectar.

**Independent Test**: Com o app em modo avião por alguns minutos e a conexão restaurada, todos os pontos capturados no período offline aparecem no servidor com o horário real de captura preservado.

### Tests for User Story 4 ⚠️

- [ ] T053 [P] [US4] Teste de integração: `POST /posicoes` insere o lote em `posicao` e só atualiza `posicao_atual` se `capturado_em` for mais recente que o já gravado, em `backend/internal/posicao/service_test.go`
- [ ] T054 [P] [US4] Teste unitário do filtro de deslocamento/intervalo (novo ponto a cada 200m ou 30s, o que ocorrer primeiro) em `app/test/localizacao_service_test.dart`

### Implementation for User Story 4

- [ ] T055 [P] [US4] Implementar repositório de `Posicao` e `PosicaoAtual` (insert em lote, update condicional por `capturado_em`) em `backend/internal/posicao/repositorio.go`
- [ ] T056 [US4] Implementar serviço de ingestão de lote de posições em `backend/internal/posicao/service.go` (depende de T055)
- [ ] T057 [US4] Implementar handler `POST /posicoes` em `backend/internal/transporte/posicoes_handler.go` (depende de T056)
- [ ] T058 [P] [US4] Implementar fila offline local com Hive (armazenar, listar, remover por confirmação) em `app/lib/data/fila_posicoes_local.dart`
- [ ] T059 [US4] Implementar `localizacao_service.dart`: coleta via `geolocator` com filtro de 200m/30s, grava na fila local, envia em lote a cada 1–2 minutos e remove apenas o confirmado pelo servidor, em `app/lib/services/localizacao_service.dart` (depende de T058)
- [ ] T060 [US4] Implementar `background_service.dart` (foreground service Android + background modes iOS) mantendo WebSocket e coleta de localização ativos com o app em segundo plano, em `app/lib/services/background_service.dart` (depende de T027, T059)
- [ ] T061 [P] [US4] Registrar dispositivo (plataforma, versão do app, `push_token`) no momento do login, em `backend/internal/usuario/` e `app/lib/features/auth/` (depende de T041)

**Checkpoint**: User Stories 1–4 funcionam de forma independente e integrada.

---

## Phase 7: User Story 5 - Painel da empresa com mapa e trajeto (Priority: P3)

**Goal**: A empresa vê no painel web a posição atual dos seus motoristas num mapa e consulta o trajeto de um motorista num período.

**Independent Test**: Com posições de teste já registradas para motoristas de empresas diferentes, um admin da empresa A só vê no mapa os motoristas da empresa A; ao consultar o trajeto de um motorista, vê o caminho percorrido no período.

### Tests for User Story 5 ⚠️

- [ ] T062 [P] [US5] Teste de integração: `GET /empresas/{id}/posicoes-atuais` retorna 403 quando `{id}` não é a empresa do admin autenticado, em `backend/internal/posicao/service_test.go`
- [ ] T063 [P] [US5] Teste de integração: `GET /motoristas/{id}/trajeto` retorna o trajeto no período informado e uma lista vazia (sem erro) quando não há dados, em `backend/internal/posicao/service_test.go`

### Implementation for User Story 5

- [ ] T064 [US5] Implementar consulta de posições atuais por empresa em `backend/internal/posicao/service.go` (depende de T055)
- [ ] T065 [US5] Implementar consulta de trajeto por motorista e período em `backend/internal/posicao/service.go` (depende de T055)
- [ ] T066 [US5] Implementar handlers `GET /empresas/{id}/posicoes-atuais` e `GET /motoristas/{id}/trajeto` em `backend/internal/transporte/posicoes_handler.go` (depende de T064, T065)
- [ ] T067 [US5] Implementar página do painel com mapa Leaflet mostrando a posição atual de cada motorista da empresa em `backend/painel/mapa.html` (depende de T066)
- [ ] T068 [US5] Implementar consulta de trajeto no painel (seleção de motorista e período, linha do percurso no mapa) em `backend/painel/trajeto.html` (depende de T066)

**Checkpoint**: Todas as 5 user stories funcionam de forma independente e integrada — v1 funcionalmente completa.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [ ] T069 [P] Enviar notificação push ao motorista quando removido por geocerca, usando `dispositivo.push_token`, em `backend/internal/geofence/verificador.go` (depende de T050, T061)
- [ ] T070 [P] Revisar `.env.example` para cobrir todas as variáveis introduzidas nas fases anteriores
- [ ] T071 Rodar quickstart.md e validar o critério de aceite da Etapa 1 de ponta a ponta (`docker compose up`, `GET /health`)
- [ ] T072 Auditoria final: confirmar que nenhuma tabela, log ou arquivo em disco armazena áudio, conforme o Princípio I da constituição
- [ ] T073 [P] Avaliar particionamento mensal de `posicao` (opcional — só após todas as etapas obrigatórias concluídas, ver data-model.md)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: sem dependências — pode começar imediatamente
- **Foundational (Phase 2)**: depende da conclusão do Setup — BLOQUEIA todas as user stories
- **User Stories (Phase 3+)**: todas dependem da conclusão da fase Foundational
  - US1 e US2 têm ambas prioridade P1; recomenda-se US1 primeiro (é o MVP mínimo demonstrável), mas ambas podem avançar em paralelo se houver mais de uma pessoa
  - US3 e US4 (P2) estendem o `entrar_canal` e a base de posições criadas por US1/US2 — melhor após elas
  - US5 (P3) depende dos dados que US4 produz (`posicao`/`posicao_atual`)
- **Polish (Phase 8)**: depende de todas as user stories desejadas estarem completas

### User Story Dependencies

- **User Story 1 (P1)**: pode começar após a Foundational. Não depende de outra story para ser testável (com dados inseridos diretamente no banco).
- **User Story 2 (P1)**: pode começar após a Foundational. Estende `entrar_canal` (T038 modifica o hub criado em T020/T022) e adiciona as telas de gestão que tornam US1 utilizável por usuários reais (não só por fixtures).
- **User Story 3 (P2)**: depende de T020/T022/T037 existirem (hub e serviço de canal), pois estende `entrar_canal` novamente com a checagem de geocerca.
- **User Story 4 (P2)**: independente das demais no backend (só usa `usuario`), mas o app reaproveita o `canal_service.dart` de US1 no `background_service.dart` (T060).
- **User Story 5 (P3)**: depende do repositório de posições criado em US4 (T055).

### Within Each User Story

- Testes (quando presentes) são escritos e devem falhar antes da implementação
- Modelos/repositórios antes de serviços
- Serviços antes de handlers HTTP/WebSocket
- Backend antes da tela do app que o consome, quando há dependência direta de dado

### Parallel Opportunities

- Todas as tarefas `[P]` da mesma fase podem rodar em paralelo (arquivos diferentes, sem dependência entre si)
- Após a Foundational, US1 e US2 podem ser desenvolvidas em paralelo por pessoas diferentes (ambas P1); US3 e US4 idem, uma vez que suas dependências estejam prontas

---

## Parallel Example: User Story 1

```bash
# Testes de US1 em paralelo:
Task: "Teste unitário: fila do hub recusa 11ª mensagem com fila_cheia em backend/internal/canal/hub_test.go"
Task: "Teste unitário: fila reproduz em FIFO e zera buffer em backend/internal/canal/hub_test.go"
Task: "Teste unitário: corte automático em 90s em backend/internal/canal/hub_test.go"
Task: "Teste de integração do protocolo WebSocket em backend/internal/ws/handler_test.go"

# Serviços de app de US1 em paralelo (após os testes falharem como esperado):
Task: "Implementar audio_service.dart em app/lib/services/audio_service.dart"
Task: "Implementar canal_service.dart em app/lib/services/canal_service.dart"
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Completar Phase 1 (Setup) e Phase 2 (Foundational — bloqueia tudo)
2. Completar Phase 3 (US1)
3. **PARAR e VALIDAR**: testar US1 de forma independente (dois motoristas de teste conversando num canal)
4. Demonstrar o MVP: o "loop de rádio" já funciona, mesmo sem telas de gestão

### Entrega Incremental

1. Setup + Foundational → fundação pronta (critério de aceite da Etapa 1)
2. US1 → validar independentemente → demonstrar o loop de voz
3. US2 → validar independentemente → canais reais, geridos pela empresa, com autorização multiempresa
4. US3 → validar independentemente → geocerca funcionando
5. US4 → validar independentemente → localização em segundo plano e fila offline
6. US5 → validar independentemente → painel da empresa com mapa e trajeto
7. Polish → notificações, auditoria de não-persistência de áudio, validação final do quickstart

Esta ordem de entrega corresponde, na prática, às 8 etapas do briefing original
da KTM (fundação → auth/canais → hub/WS → áudio no app → localização → painel
→ geocerca/segundo plano), apenas reorganizadas por user story para permitir
teste e demonstração independentes a cada incremento — sem contradizer o
Princípio II da constituição (uma etapa por vez, com aceite validado antes de
seguir): cada checkpoint acima é o ponto de parada para apresentar o critério
de aceite e aguardar confirmação antes de avançar.

---

## Notes

- `[P]` = arquivos diferentes, sem dependência entre as tarefas
- `[Story]` mapeia a tarefa à user story correspondente, para rastreabilidade
- Cada user story deve ser completável e testável de forma independente
- Verificar que os testes falham antes de implementar
- Commitar após cada tarefa ou grupo lógico de tarefas (Princípio "commits pequenos e descritivos" da constituição)
- Parar em qualquer checkpoint para validar a story de forma independente antes de seguir
- Evitar: tarefas vagas, conflitos no mesmo arquivo, dependências entre stories que quebrem a independência
