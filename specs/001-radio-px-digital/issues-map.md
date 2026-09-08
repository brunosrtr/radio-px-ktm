# Mapa Issues do GitHub ↔ tasks.md

Feature: Rádio PX Digital v1 (`001-radio-px-digital`)

As 18 issues originais do repositório (`brunosrtr/radio-px-ktm`) foram criadas
por requisito de negócio (RF/RNF), antes do plano técnico existir. O
`tasks.md` desta feature é mais granular (uma tarefa por passo de
implementação). Esta tabela alinha as duas coisas para que os commits de
implementação referenciem a issue certa.

**Convenção de commit**: usar `Refs #N` nos commits que avançam parcialmente
uma issue, e `Closes #N` no commit (ou PR) que satisfaz o último critério de
aceite pendente daquela issue — o GitHub fecha a issue automaticamente.

## Issues fechadas (fora de escopo da v1)

| Issue | Motivo |
|---|---|
| #4 — Integração com o hardware Orbit | Depende do Orbit; v1 usa apenas internet móvel (ver spec.md) |
| #5 — Geocerca via localização do Orbit | Reescrito para GPS do celular; ver #19 |

## Issues novas (cobrindo lacunas de US4/US5)

| Issue | Título | User Story | RF/RNF |
|---|---|---|---|
| #19 | Restrição de acesso a canal por geocerca via GPS do dispositivo | US3 | FR-014, FR-015 |
| #20 | Coleta periódica de localização com filtro de deslocamento e fila offline | US4 | FR-024, FR-028 |
| #21 | Envio em lote de posições e retenção de histórico | US4 | FR-025, FR-033 |
| #22 | Painel web: mapa com posição atual dos motoristas | US5 | FR-026 |
| #23 | Painel web: consulta de trajeto por motorista e período | US5 | FR-027 |

## Mapa completo issue → tasks.md

| Issue | Título | User Story | Tasks relacionadas |
|---|---|---|---|
| #1 | Transmissão de voz em tempo real | US1 | T019–T025 (hub + WS), T026–T028 (app) |
| #2 | Fila de transmissão de mensagens de voz | US1 | T015–T017 (testes), T019–T021 (hub) |
| #3 | Exclusão permanente após reprodução | US1 | T016 (teste zera buffer), T020, T072 (auditoria final) |
| #6 | Remoção automática ao sair da área | US3 | T047 (teste), T050 (verificador) |
| #7 | Execução em segundo plano (áudio e localização) | US1 + US4 | T060 (background_service.dart) |
| #8 | Cadastro e gerenciamento de canais pela empresa | US2 | T034, T035, T037, T040 |
| #9 | Limite configurável de participantes | US2 | T031 (teste), T038 |
| #10 | Canais compartilhados entre empresas | US2 | T035, T040 |
| #11 | Autenticação e autorização de acesso a canais | Foundational + US2 | T010, T011, T041, T043, T045 |
| #12 | Gravação e envio com limite de 1min30 | US1 | T017 (teste), T021, T026 |
| #13 | Setup do app para Android e iOS | Setup | T001, T003 |
| #14 | Listagem e alternância entre canais | US2 | T039, T044 |
| #15 | Saída manual do canal | US2 | T039 |
| #16 | Silenciar canal sem sair dele | US2 | T036, T040 |
| #17 | Avisos: fila cheia e remoção por área | US1 + US3 | T028 (fila cheia), T052 (aviso geocerca) |
| #18 | Identificação do motorista na transmissão | US1 | T024, T028 |
| #19 | Geocerca via GPS (acesso) | US3 | T048, T049, T051 |
| #20 | Coleta periódica + fila offline | US4 | T058, T059 |
| #21 | Envio em lote + retenção | US4 | T053, T055, T056, T057 |
| #22 | Painel: mapa de posição atual | US5 | T064, T066, T067 |
| #23 | Painel: trajeto por período | US5 | T065, T066, T068 |

## Sem issue própria (infraestrutura, não é requisito de negócio)

Tarefas de Setup/Foundational que não representam um requisito RF/RNF
específico (T002, T004–T009, T012–T014, T032–T033, T042, T061,
T069–T071, T073) não precisam de issue — ficam apenas rastreadas pelo
`tasks.md` e podem ser referenciadas diretamente pelo ID da task no commit
(ex.: `T007`), sem número de issue.
