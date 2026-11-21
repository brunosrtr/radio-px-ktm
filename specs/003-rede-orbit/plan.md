# Implementation Plan: Rede Orbit — Rede Aberta de Caminhoneiros e Retransmissão Colaborativa de Sinais Orbit

**Branch**: `003-rede-orbit` | **Date**: 2026-11-21 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-rede-orbit/spec.md`

## Summary

O app passa a ser aberto ao público e cada celular vira um **ponto de transmissão**. Ele busca por
Bluetooth Low Energy o sinal do Orbit, filtrando por um Service UUID fixo, deduplica por Orbit a cada
30 s, guarda numa fila Hive offline e envia lotes de leituras ao backend Go. O backend grava em SQL
escrito à mão, atualiza a última posição de cada Orbit de forma condicional e encaminha as leituras
por uma interface `Integracao` substituível. Na demo, essa interface alimenta o painel **Rede Orbit**
ao vivo (Leaflet + WebSocket).

Para a demonstração sem hardware, um celular Android em **modo Orbit virtual** anuncia esse mesmo
sinal. Em paralelo:
- contas passam a ser de cadastro próprio (empresa opcional);
- canais passam a ser criados pela comunidade (públicos ou privados por convite);
- o app ganha nova navegação em 4 abas, apresentação inicial e permissões explicadas.

A rádio da 001 (hub em memória, fila FIFO, áudio nunca persistido) não muda.

## Technical Context

**Language/Version**: Go (backend + painel), Dart/Flutter (app), sem mudança em relação à 001.
**Primary Dependencies**: as da 001, mais:
- app: `flutter_blue_plus` (leitura BLE), `flutter_ble_peripheral` (Orbit virtual), `go_router` (navegação/deep link), `qr_flutter` (gerar QR do convite);
- backend: `net/smtp` da biblioteca padrão (recuperação de senha); nenhuma dependência Go nova.

Ver [research.md](./research.md) §2, §7 e §10.

**Storage**: PostgreSQL, com a migração `0004_rede_orbit.sql` ([data-model.md](./data-model.md)). Hive no app ganha a caixa `fila_leituras`. Os pontos ativos ficam só em memória.

**Testing**: `go test` com Postgres real (como na 001) e `flutter test`. Testes obrigatórios novos, pelo Princípio IV ampliado:
- `DeduplicadorLeituras`;
- ingestão em lote de leituras com atualização condicional de `orbit.ultima_*` e idempotência no reenvio;
- `canal.PodeEntrar`;
- parser do sinal Orbit.

**Target Platform**: Android 8+ e iOS 15+ (app); Linux em container (backend/painel). Orbit virtual: só Android com suporte a anúncio BLE.

**Project Type**: app móvel + API com painel web embutido (igual à 001).

**Performance Goals**:
- leitura visível no painel em ≤ 5 s com o app em primeiro plano (SC-001);
- ≥ 95% de detecção de um Orbit que fica 1 min no alcance com o app em segundo plano no Android (SC-002);
- latência de voz da 001 mantida.

**Constraints**:
- áudio nunca persiste (Princípio I);
- painel sem dados pessoais e app sem dados do Orbit (Princípio VII);
- histórico de posições só para quem tem empresa (FR-034);
- consumo de bateria ≤ +5 p.p./h com a função ativa (SC-007);
- envio só em lotes (FR-004).

**Scale/Scope**: demonstração com 3 celulares e 1 painel. O projeto deve suportar centenas de pontos de transmissão numa única instância (cerca de 1 lote a cada 15 s por ponto ativo, ≤ 100 leituras por lote).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Referência: [constitution.md](/.specify/memory/constitution.md) v2.0.0.

| Princípio | Gate | Status |
|---|---|---|
| I. Áudio nunca é persistido | Nenhuma tabela nova ou caixa Hive nova contém áudio; o hub de voz não é alterado por esta feature. Ao apagar um canal, os participantes são desconectados sem preservar o buffer. | PASS |
| II. Uma etapa por vez | 8 etapas sequenciais abaixo, cada uma com critério de aceite; a Etapa 4 (protótipo de UX) exige aprovação explícita antes da 5. | PASS |
| III. Nada de requisito inventado | Spec sem `[NEEDS CLARIFICATION]` (FR-033/034 decididos em 2026-11-21). O protocolo do Orbit real segue como Pendência; o formato do Orbit virtual é declaradamente provisório ([sinal-orbit.md](./contracts/sinal-orbit.md)). | PASS |
| IV. Testes das regras críticas | Testes da 001 mantidos, mais os 4 testes obrigatórios novos listados em Testing. | PASS (reforçado em tasks.md) |
| V. Dados explícitos, sem ORM | Migração 0004 em SQL idempotente; repositórios novos com `pgx`. | PASS |
| VI. Simplicidade | Instância única; pontos ativos e limitador de cadastro em memória; `orbit.Integracao` e `orbit_sinal.dart` são os pontos de troca exigidos. | PASS |
| VII. Privacidade em mão dupla | `leitura_orbit` usa `ponto_id` opaco e rotativo (HMAC por dia), nunca `usuario_id`; contadores pessoais ficam em `contribuicao_diaria`, que o painel não lê; o app não recebe dados de Orbit; consentimento `ponto_transmissao` separado e revogável. | PASS |
| Segurança e Configuração | Rotas públicas restritas a cadastro, login e recuperação de senha (mais os textos de termos e privacidade); novos segredos (`PONTO_ID_SEGREDO`, `SMTP_*`, `KTM_INTEGRACAO_URL`) entram no `.env.example`. | PASS |

**Re-check pós-design (Fase 1)**: data-model e contratos conferidos contra os gates acima. Nenhuma violação; Complexity Tracking vazio.

## Etapas de implementação

Cada etapa só começa após o aceite da anterior (Princípio II). As Etapas 1 a 3 entregam o **marco da demonstração**.

| # | Etapa | Entrega | Critério de aceite |
|---|---|---|---|
| 1 | **Contas abertas** | Migração 0004 (parte de contas), cadastro, login por e-mail ou login, recuperação de senha com Mailpit, consentimentos, exclusão de conta, JWT com `empresa_id` opcional; telas simples de cadastro e recuperação no app | Criar conta pelo app, sair, recuperar a senha pelo Mailpit e entrar de novo; contas antigas de empresa continuam entrando; `go test` verde |
| 2 | **Rede Orbit, backend e painel** | Tabelas `orbit`, `leitura_orbit`, `ponto_fixo`, `contribuicao_diaria`; `POST /leituras`, `/ponto/sinal`, `/me/contribuicao`; `orbit.Integracao` + `IntegracaoPainel`; `/ws/rede`; `painel/rede.html`; seed `-demo` | Um lote enviado por `curl` aparece no mapa em < 5 s; reenviar o mesmo lote não duplica; uma leitura antiga não move o Orbit; testes obrigatórios verdes |
| 3 | **Ponto de transmissão e Orbit virtual no app** | `orbit_sinal.dart`, `LeitorOrbit` (busca no foreground service), `DeduplicadorLeituras`, `fila_leituras`, bip, modo Orbit virtual (`admin_ktm`/`demo`), permissões BLE em Android/iOS, modo leitura do `LocalizacaoService` para quem não tem empresa | **Roteiro do quickstart, passos 3 a 6, em aparelhos reais**; `flutter test` verde |
| 4 | **Protótipo de UX** | Protótipo navegável das telas: apresentação inicial, permissões, 4 abas, canal ativo, criar canal, convite, contribuição, perfil | Aprovação explícita do usuário |
| 5 | **Nova navegação e experiência** | `go_router`, `NavigationBar` em 4 abas, apresentação inicial, tela de permissões, novo canal ativo, tema claro e escuro, fontes do sistema | Teste com 3 pessoas que nunca usaram o app (SC-004/SC-005) |
| 6 | **Canais da comunidade** | Migração 0004 (parte de canais), `PodeEntrar`, explorar (busca e próximos), convite por código e QR com deep link, favoritos, banir, apagar canal | Criar canal privado, entrar por QR em < 1 min (SC-006); testes de `PodeEntrar` verdes |
| 7 | **Contribuição e privacidade** | Aba Contribuição, textos de termos e privacidade, revogar consentimento, auditoria do Princípio VII (app e painel) | Checklist SC-008 conferido tela a tela |
| 8 | **Preparação da demo** | Seed final, roteiro ensaiado, vídeo de plano B, medição de bateria (SC-007) | Roteiro completo executado sem falhas 2 vezes seguidas |

## Project Structure

### Documentation (this feature)

```text
specs/003-rede-orbit/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── rest-api.md
│   ├── sinal-orbit.md
│   └── painel-rede-ws.md
├── checklists/requirements.md
└── tasks.md             # gerado por /speckit.tasks
```

### Source Code (repository root)

Só o que é novo ou alterado. O resto segue a estrutura da 001.

```text
backend/
├── migrations/0004_rede_orbit.sql          # novo
├── cmd/seed/main.go                         # + flag -demo
├── internal/
│   ├── auth/                                # empresa_id opcional, novos papéis
│   ├── usuario/
│   │   ├── cadastro_publico.go              # novo: cadastro, recuperação, exclusão
│   │   └── consentimento.go                 # novo
│   ├── email/                               # novo: envio SMTP (recuperação)
│   ├── canal/
│   │   ├── acesso.go                        # novo: PodeEntrar (função pura)
│   │   ├── convite.go                       # novo
│   │   └── service.go / repositorio.go      # canais da comunidade
│   ├── orbit/                               # novo pacote
│   │   ├── leitura.go                       # entidades e validação do lote
│   │   ├── repositorio.go                   # ingestão + atualização condicional
│   │   ├── service.go                       # orquestra ingestão, contribuição, integração
│   │   ├── integracao.go                    # interface + IntegracaoPainel + IntegracaoHTTP (esqueleto)
│   │   ├── pontos_ativos.go                 # mapa em memória
│   │   └── ponto_id.go                      # HMAC rotativo
│   ├── transporte/
│   │   ├── auth_handler.go                  # + cadastro, recuperação
│   │   ├── leituras_handler.go              # novo
│   │   └── rede_handler.go                  # novo (painel admin_ktm)
│   └── ws/rede_handler.go                   # novo
└── painel/rede.html                         # novo

app/lib/
├── core/
│   ├── router.dart                          # novo (go_router), substitui routes.dart
│   └── theme.dart                           # tema claro/escuro
├── orbit/                                   # novo
│   ├── orbit_sinal.dart                     # ÚNICO ponto do formato do sinal (FR-016)
│   ├── leitor_orbit.dart                    # busca BLE
│   ├── deduplicador_leituras.dart           # puro, testado
│   ├── orbit_virtual.dart                   # anúncio BLE (demonstração)
│   └── leituras_service.dart                # fila + envio em lote + bip
├── data/fila_leituras_local.dart            # novo
├── services/
│   ├── background_service.dart              # + busca BLE no foreground service
│   └── localizacao_service.dart             # + modo leitura (sem histórico)
└── features/
    ├── onboarding/                          # novo: apresentação + permissões
    ├── auth/                                # + cadastro, recuperação
    ├── canais/                              # + explorar, criar, convite
    ├── contribuicao/                        # novo
    ├── perfil/                              # novo (inclui modo Orbit virtual p/ demo)
    └── shell/                               # novo: NavigationBar com 4 abas
```

**Structure Decision**: mantém os dois componentes da 001 (`backend/` e `app/`). A lógica da Rede Orbit fica num pacote próprio em cada lado (`internal/orbit/` e `lib/orbit/`). Assim o núcleo novo do TCC fica isolado e legível, e o formato provisório do sinal e a integração com a KTM ficam cada um num único ponto de troca (Princípio VI).

## Complexity Tracking

Nenhuma violação da constituição a justificar.
