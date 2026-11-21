# Tasks: Rede Orbit — Rede Aberta de Caminhoneiros e Retransmissão Colaborativa de Sinais Orbit

**Input**: Design documents from `/specs/003-rede-orbit/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Organização**: as fases seguem as **8 etapas** do [plan.md](./plan.md), porque o Princípio II exige uma etapa por vez com aceite. Cada tarefa leva a user story da [spec.md](./spec.md) que atende. **Só a Etapa 1 está detalhada.** As demais aparecem como resumo e são detalhadas quando a etapa anterior for aceita, para que o detalhamento use o código como ele realmente ficou.

**Testes**: obrigatórios para as regras do Princípio IV (marcados com 🔒). Também entram testes de cadastro e recuperação de senha, que são fluxos públicos sensíveis a abuso.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: pode rodar em paralelo (arquivos diferentes, sem dependência de tarefa incompleta)
- **[Story]**: US1–US7 da spec (US1 ponto de transmissão, US2 Orbit virtual, US3 painel, US4 cadastro aberto, US5 canais da comunidade, US6 UX, US7 contribuição)

---

## Etapa 1 — Contas abertas (US4)

**Objetivo**: qualquer pessoa cria conta pelo app, recupera a senha e exclui a conta. Contas de empresa da 001 continuam funcionando sem mudança.

**Critério de aceite**: criar conta pelo app → sair → recuperar a senha pelo Mailpit → entrar de novo. Um motorista antigo continua entrando com o login de sempre. `go test ./...` e `flutter test` verdes.

**Fora desta etapa** (não é regressão): o usuário do público entra e vê a lista de canais **vazia**. Os canais da comunidade chegam na Etapa 6, e a apresentação inicial e as permissões na Etapa 5.

### Configuração

- [X] T001 [P] Adicionar o serviço `mailpit` (SMTP 1025, web 8025) em `docker-compose.yml` e passar `SMTP_*` para o serviço `backend`
- [X] T002 [P] Adicionar `SMTP_HOST`, `SMTP_PORTA`, `SMTP_USUARIO`, `SMTP_SENHA`, `SMTP_REMETENTE` em `.env.example`, comentadas e com valores do Mailpit
- [X] T003 Ler `SMTP_*` (opcionais) em `backend/internal/config/config.go`; `Config.SMTPConfigurado()` retorna se o host está definido

### Base de dados e autorização (bloqueia o resto da etapa)

- [X] T004 Criar `backend/migrations/0004_contas_abertas.sql` conforme [data-model.md §1](./data-model.md):
  - extensão `citext`;
  - em `usuario`: `email`, `apelido` e `excluido_em` novos, `empresa_id`/`login` aceitando nulo, check de `papel` ampliado, checks de coerência papel↔empresa e email/login;
  - preenchimento de `apelido` nas contas existentes;
  - tabelas `recuperacao_senha` e `consentimento`;
  - tudo idempotente: `if not exists` e blocos `do $$` para as constraints
- [X] T005 Atualizar `backend/internal/usuario/usuario.go`:
  - campos `Email`, `Apelido`, `ExcluidoEm`;
  - `empresa_id` e `login` lidos com `coalesce(...::text, '')`, para que `""` signifique "sem vínculo" sem mudar o tipo `string` usado nos outros pacotes;
  - `Criar` grava com `nullif($n, '')::uuid`;
  - novas constantes `PapelUsuario`, `PapelAdminKTM`, `PapelDemo`
- [X] T006 Adicionar `BuscarPorIdentificador` em `backend/internal/usuario/usuario.go` (e-mail sem diferenciar maiúsculas de minúsculas **ou** login; só contas ativas e não excluídas)
- [X] T007 Garantir que empresa vazia nunca autoriza canal de empresa:
  - em `backend/internal/canal/service.go` (`ObterParaEntrada`, `ListarParaMotorista`) e `backend/internal/ws/central_handler.go`, retornar `ErrSemPermissao` / lista vazia quando `empresaID == ""`;
  - acrescentar o método `TemEmpresa()` às `Claims` em `backend/internal/auth/auth.go`
- [X] T008 🔒 Teste em `backend/internal/canal/service_test.go`: um usuário sem empresa não lista nem entra em canal privado ou compartilhado de empresa
- [X] T009 Recusar `POST /posicoes` de usuário sem empresa com `403 sem_historico` (FR-034) em `backend/internal/transporte/posicoes_handler.go`, com teste em `backend/internal/transporte/posicoes_handler_test.go`

### Backend: cadastro, recuperação, consentimento e exclusão

- [X] T010 [P] [US4] Criar o pacote `backend/internal/email/` (`email.go`):
  - interface `Enviador` com `Enviar(ctx, para, assunto, corpo)`;
  - `EnviadorSMTP` (`net/smtp`);
  - `EnviadorMemoria` para testes;
  - sem SMTP configurado, `NovoEnviador` retorna `nil`
- [X] T011 [P] [US4] Criar `backend/internal/usuario/cadastro_publico.go`:
  - `DadosCadastroPublico`, `ValidarCadastroPublico` (nome obrigatório, apelido 2–24, e-mail válido normalizado, senha ≥ 8, versão de termos obrigatória) e `CriarUsuarioPublico` (papel `usuario`, sem empresa);
  - na mesma transação, grava o consentimento `termos`;
  - `ErrEmailEmUso` vem da violação de unicidade
- [X] T012 [P] [US4] Criar `backend/internal/usuario/consentimento.go` com `RegistrarConsentimento`, `RevogarConsentimento` e `ConsentimentosVigentes` (tipos `termos` e `ponto_transmissao`)
- [X] T013 [P] [US4] Criar `backend/internal/usuario/recuperacao.go`:
  - `SolicitarRecuperacao`: código de 6 dígitos via `crypto/rand`, hash bcrypt, validade de 15 min; invalida os códigos anteriores do usuário; envia pelo `email.Enviador`; para e-mail inexistente, não faz nada e não dá erro;
  - `RedefinirSenha`: no máximo 5 tentativas, uso único, troca o `senha_hash`
- [X] T014 [P] [US4] Criar `backend/internal/usuario/exclusao.go` com `ExcluirConta`. Numa transação:
  - anonimiza `nome`, `apelido`, `email`, `login`, `cpf`;
  - `ativo = false` e `excluido_em = now()`;
  - apaga `posicao_atual`, `dispositivo`, `preferencia_canal` e `recuperacao_senha`;
  - encerra as participações abertas com motivo `desconexao`
- [X] T015 [P] [US4] Criar o limitador em memória `backend/internal/transporte/limitador.go` (janela deslizante por IP, configurável; padrão de 5 a cada 10 min), sem dependência externa (Princípio VI)
- [X] T016 [US4] Estender `backend/internal/transporte/auth_handler.go` conforme [contracts/rest-api.md](./contracts/rest-api.md):
  - `Cadastro`;
  - `Login`, que aceita `identificador`, ou `login` por compatibilidade com o app atual, e devolve `apelido`;
  - `RecuperarSenha`: `202` sempre, `503` sem SMTP;
  - `RedefinirSenha`;
  - `Me`, com `apelido`, `email`, `empresa_id` nulo quando não há empresa, e consentimentos;
  - `AtualizarMe`, `ExcluirMe` e `DefinirConsentimento`
- [X] T017 [P] [US4] Criar `backend/painel/termos.html` e `backend/painel/privacidade.html` (versão `2026-11`), textos simples que citam o Princípio VII, servidos publicamente
- [X] T018 [US4] Registrar em `backend/internal/transporte/router.go`:
  - rotas públicas `POST /auth/cadastro`, `/auth/recuperar-senha` e `/auth/redefinir-senha`, as duas primeiras com o limitador;
  - `GET /termos` e `/privacidade`;
  - rotas autenticadas `PATCH /me`, `DELETE /me` e `PUT /me/consentimentos/{tipo}`;
  - ligar o `email.Enviador` em `backend/cmd/api/main.go`
- [X] T019 [US4] Usar o apelido como nome de quem fala no rádio (FR-027): em `backend/internal/ws/handler.go`, `nome: u.Apelido`, com `u.Nome` como reserva se o apelido estiver vazio

### Backend: testes

- [X] T020 [P] [US4] `backend/internal/usuario/cadastro_publico_test.go`:
  - casos de validação;
  - e-mail duplicado sem diferenciar maiúsculas (`Maria@x.com` × `maria@x.com`) → `ErrEmailEmUso`;
  - consentimento `termos` gravado;
  - conta de empresa antiga continua sendo encontrada por `BuscarPorIdentificador` pelo login
- [X] T021 [P] [US4] `backend/internal/usuario/recuperacao_test.go`:
  - código certo troca a senha;
  - código expirado, já usado ou na 6ª tentativa é recusado;
  - um novo pedido invalida o código anterior;
  - e-mail inexistente não envia nada (verificado com `EnviadorMemoria`)
- [X] T022 [P] [US4] `backend/internal/usuario/exclusao_test.go`: depois de excluir, o login pelo e-mail antigo falha, os dados pessoais estão anonimizados e o mesmo e-mail pode criar uma conta nova
- [X] T023 [P] [US4] `backend/internal/transporte/auth_handler_test.go`, testes de contrato:
  - `201`/`409`/`422`/`429` no cadastro;
  - login por e-mail e por login;
  - corpo antigo `{"login": ...}` ainda aceito;
  - `202` na recuperação para e-mail inexistente;
  - `401` em `DELETE /me` sem token
- [X] T024 [P] [US4] `backend/internal/transporte/limitador_test.go`: a 6ª tentativa na janela é recusada e libera depois que a janela passa (com relógio injetável)

### App

- [X] T025 [US4] Estender `app/lib/features/auth/auth_service.dart`:
  - `login(identificador, senha)`;
  - `cadastrar(...)`;
  - `solicitarRecuperacao(email)` e `redefinirSenha(email, codigo, novaSenha)`;
  - `excluirConta()`;
  - `me()`, que devolve um `UsuarioSessao` com `apelido`, `papel`, `temEmpresa`, em `app/lib/models/usuario_sessao.dart`
- [X] T026 [P] [US4] Criar `app/lib/features/auth/cadastro_page.dart`:
  - campos nome, apelido de rádio, e-mail, senha e confirmação;
  - caixa "Li e aceito os termos de uso e a política de privacidade", com links para `/termos` e `/privacidade`;
  - validação na própria tela e mensagens para `email_em_uso`, `muitas_tentativas` e `dados_invalidos`;
  - identidade visual atual (`core/theme.dart`)
- [X] T027 [P] [US4] Criar `app/lib/features/auth/recuperar_senha_page.dart` em dois passos: informar o e-mail, depois código + nova senha. Mostra a mensagem neutra "se o e-mail tiver conta, enviamos um código"
- [X] T028 [US4] Ajustar `app/lib/features/auth/login_page.dart`:
  - campo "E-mail ou login";
  - botões "Criar conta" e "Esqueci minha senha";
  - trocar o comentário e textos que dizem que as contas são fornecidas pela empresa
- [X] T029 [US4] Registrar as rotas `cadastro` e `recuperarSenha` em `app/lib/core/routes.dart` e `app/lib/main.dart`. O `go_router` só entra na Etapa 5
- [X] T030 [US4] Não iniciar o histórico de posições para quem não tem empresa (FR-034): em `app/lib/features/canais/canal_ativo_page.dart`, chamar `_localizacaoService.iniciar()` só quando `UsuarioSessao.temEmpresa`. O modo leitura completo fica para a Etapa 3
- [X] T031 [P] [US4] Criar a tela provisória `app/lib/features/configuracoes/conta_page.dart` com apelido, e-mail, "Sair" e "Excluir conta" (com confirmação), acessível pelo menu da lista de canais. Ela vira a aba Perfil na Etapa 5
- [X] T032 [P] [US4] `app/test/auth_service_test.dart`: `ApiClient` simulado; cadastro envia os campos do contrato, guarda o token e mapeia `409` para `EmailEmUsoException`; login envia `identificador`
- [X] T033 [P] [US4] `app/test/cadastro_page_test.dart` (widget): botão desabilitado sem aceite dos termos; senhas diferentes mostram erro; apelido com 1 caractere mostra erro

### Fechamento da etapa

- [X] T034 Atualizar o seed `backend/cmd/seed/main.go` para preencher `apelido` nas contas de exemplo
- [ ] T035 Rodar `cd backend && go test ./...`, `cd app && flutter analyze && flutter test`, e o critério de aceite em um celular real com o Mailpit

**Checkpoint**: parar e pedir o aceite do usuário (Princípio II).

### Dependências da Etapa 1

- T001–T003 → T004 → T005/T006 → T007–T009 e T010–T015 (os [P] em paralelo) → T016 → T018 → T019
- Testes T020–T024 logo após as respectivas implementações
- App (T025–T033) depende só do contrato. Pode começar junto com o backend, mas T035 exige os dois prontos

---

## Etapa 2 — Rede Orbit: backend e painel (US3, base de US1/US2) · *a detalhar*

Migração `0005_rede_orbit.sql`; pacote `backend/internal/orbit/` (validação do lote, repositório com ingestão idempotente e atualização condicional 🔒, `ponto_id` HMAC rotativo, pontos ativos em memória, `Integracao` + `IntegracaoPainel`); `POST /leituras`, `/ponto/sinal`, `/me/contribuicao`; rotas `/rede/*` e `/ws/rede` para `admin_ktm`; `backend/painel/rede.html`; seed `-demo`; `PONTO_ID_SEGREDO` no `.env.example`.

## Etapa 3 — Ponto de transmissão e Orbit virtual no app (US1, US2) · *a detalhar*

`app/lib/orbit/` (`orbit_sinal.dart` com parser 🔒, `deduplicador_leituras.dart` 🔒, `leitor_orbit.dart`, `orbit_virtual.dart`, `leituras_service.dart`); `fila_leituras_local.dart`; busca BLE no foreground service; permissões e `UIBackgroundModes`; bip; consentimento `ponto_transmissao`; `PUT /posicao-atual` e modo leitura do `LocalizacaoService`.

## Etapa 4 — Protótipo de UX (US6) · *a detalhar*

## Etapa 5 — Nova navegação e experiência (US4, US6) · *a detalhar*

## Etapa 6 — Canais da comunidade (US5) · *a detalhar*

Inclui a migração `0006_canais_comunidade.sql` e `canal.PodeEntrar` 🔒.

## Etapa 7 — Contribuição e privacidade (US7) · *a detalhar*

## Etapa 8 — Preparação da demo · *a detalhar*
