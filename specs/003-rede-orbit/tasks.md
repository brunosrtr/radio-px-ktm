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

## Etapa 2 — Rede Orbit: backend e painel (US3, base de US1/US2)

**Objetivo**: o backend recebe lotes de leituras Orbit, guarda sem identificar quem leu, mantém a última posição de cada Orbit e mostra tudo ao vivo no painel Rede Orbit.

**Critério de aceite**: um lote enviado por `curl` aparece no mapa do painel em < 5 s; reenviar o mesmo lote não duplica; uma leitura mais antiga não move o Orbit; testes obrigatórios 🔒 verdes.

### Configuração e base de dados

- [X] T036 [P] Ler `PONTO_ID_SEGREDO` (opcional; sem ele, derivado de `JWT_SECRET`) e `KTM_INTEGRACAO_URL` (opcional) em `backend/internal/config/config.go`; documentar no `.env.example`
- [X] T037 Criar `backend/migrations/0005_rede_orbit.sql` com `orbit`, `leitura_orbit`, `ponto_fixo` e `contribuicao_diaria` conforme [data-model.md §4](./data-model.md)

### Pacote `backend/internal/orbit/`

- [X] T038 [P] [US1] `leitura.go`: tipos `LeituraRecebida`/`LeituraRegistrada`/`OrbitResumo`/`PontoFixo` e `ValidarLote` (1–100 leituras, `ORBIT-` + 6 hex, coordenadas válidas, `detectado_em` entre agora − 7 dias e agora + 2 min)
- [X] T039 [P] [US1] `ponto_id.go`: `PontoID(segredo, usuarioID, instante)` = HMAC-SHA256 truncado em 16 hex, renovado por dia (fuso `America/Sao_Paulo`)
- [X] T040 [P] [US3] `pontos_ativos.go`: mapa em memória `ponto_id → visto_em`, com `Registrar` e `Contar(janela)` e relógio injetável
- [X] T041 [US1] `repositorio.go`: `IngerirLote` numa transação. Ela cria os Orbits que faltam, insere as leituras com `on conflict do nothing` (idempotente), atualiza `orbit.ultima_*` **só** com leituras mais recentes e soma em `contribuicao_diaria` só as leituras novas. Também: `Orbits`, `HistoricoOrbit` e CRUD de `ponto_fixo`
- [X] T042 [P] [US3] `broadcaster.go`: eventos da Rede Orbit para os painéis (não bloqueia; um assinante lento perde o evento)
- [X] T043 [P] [US3] `integracao.go`: interface `Integracao` + `IntegracaoPainel` (publica no broadcaster) + `IntegracaoHTTP` (POST JSON para `KTM_INTEGRACAO_URL`, só ligada se configurada)
- [X] T044 [US1] [US3] `service.go`: `IngerirLeituras` (valida, calcula `ponto_id`, grava, registra ponto ativo, encaminha para as integrações sem fazer o lote falhar), `RegistrarSinal`, `Contribuicao`, `SimularPassagem` (leitura de origem `fixo`)

### HTTP e WebSocket

- [X] T045 [US1] Adicionar `TemConsentimento(ctx, usuarioID, tipo)` em `backend/internal/usuario/consentimento.go`
- [X] T046 [US1] `backend/internal/transporte/leituras_handler.go`: `POST /leituras` (`403 sem_consentimento`, `422 lote_invalido`, `200 {aceitas, duplicadas}`), `POST /ponto/sinal`, `GET /me/contribuicao`
- [X] T047 [US3] `backend/internal/transporte/rede_handler.go`: `/rede/orbits`, `/rede/orbits/{id}/leituras`, `/rede/pontos-fixos` (GET/POST/DELETE) e `/rede/pontos-fixos/{id}/simular`, todos só para `admin_ktm`
- [X] T048 [US3] `backend/internal/ws/rede_handler.go`: `/ws/rede` (`admin_ktm`), que envia `inicial`, repassa `leitura`/`orbit_atualizado` e manda `pontos_ativos` a cada 30 s
- [X] T049 Registrar as rotas e montar o serviço em `backend/internal/transporte/router.go`

### Painel e seed

- [X] T050 [US3] `backend/painel/rede.html`:
  - login de `admin_ktm`;
  - mapa Leaflet com os Orbits, pulso a cada leitura, pontos fixos como círculos e botão para mostrar ou ocultar a cobertura fixa;
  - lista de Orbits com a última leitura;
  - histórico de 24 h desenhado no mapa;
  - cadastro de ponto fixo clicando no mapa;
  - "simular passagem por ponto fixo";
  - contador de pontos ativos e feed das últimas leituras
- [X] T051 [US2] [US3] Flag `-demo` em `backend/cmd/seed/main.go`, idempotente:
  - contas `admin_ktm` e `demo`, com consentimento de ponto de transmissão;
  - 3 Orbits virtuais com rótulo;
  - 4 pontos fixos simulados

### Testes

- [X] T052 [P] 🔒 `backend/internal/orbit/leitura_test.go`, validação do lote: tamanho, formato do ID, coordenadas, horário no futuro ou antigo demais
- [X] T053 [P] 🔒 `backend/internal/orbit/repositorio_test.go`:
  - reenviar o mesmo lote não duplica;
  - leitura mais antiga não move o Orbit e leitura mais nova move;
  - contribuição conta só as leituras novas;
  - `leitura_orbit` não tem coluna que identifique o usuário
- [X] T054 [P] `backend/internal/orbit/ponto_id_test.go` e `pontos_ativos_test.go`:
  - o `ponto_id` muda de um dia para o outro e é diferente para usuários diferentes;
  - a janela de pontos ativos funciona
- [X] T055 `backend/internal/transporte/leituras_handler_test.go`:
  - contrato de `/leituras`, `/ponto/sinal`, `/me/contribuicao` e `/rede/*`;
  - `403` para quem não tem consentimento e para quem não é `admin_ktm`;
  - o WebSocket `/ws/rede` recebe `inicial` e depois a `leitura`, sem `usuario_id`, nome ou e-mail no evento (Princípio VII)
- [X] T056 Rodar `go test ./...` e o critério de aceite com `curl` + painel no navegador

**Checkpoint**: parar e pedir o aceite do usuário (Princípio II).

## Etapa 3 — Ponto de transmissão e Orbit virtual no app (US1, US2)

**Objetivo**: o celular detecta o sinal do Orbit (real ou virtual), faz o bip e envia a leitura em lote, inclusive em segundo plano e depois de ficar sem internet. Um celular Android com conta `demo` ou `admin_ktm` faz o papel do Orbit.

**Critério de aceite**: passos 3 a 6 do [quickstart.md](./quickstart.md) em aparelhos reais; `flutter test` verde.

**Decisão desta etapa**: a leitura BLE usa `flutter_reactive_ble` (BSD-3) em vez de `flutter_blue_plus`. A versão 2.x do `flutter_blue_plus` exige licença comercial para uso por empresa com fins lucrativos e envia telemetria no build ([research.md §2](./research.md)).

### Backend

- [X] T057 [US1] `PUT /posicao-atual` (só para quem não tem empresa; um ponto, sem histórico — FR-034), em `backend/internal/transporte/posicoes_handler.go` e `backend/internal/posicao/repositorio.go`, com teste

### Núcleo da leitura (puro, testado)

- [X] T058 [P] [US1] 🔒 `app/lib/orbit/orbit_sinal.dart`: UUID Rede Orbit, `codificarCarga(orbitId)` e `lerCarga(bytes)` conforme [contracts/sinal-orbit.md](./contracts/sinal-orbit.md), com teste em `app/test/orbit_sinal_test.dart`
- [X] T059 [P] [US1] 🔒 `app/lib/orbit/deduplicador_leituras.dart`: no máximo uma leitura por Orbit a cada 30 s e ignora o próprio Orbit virtual, com teste em `app/test/deduplicador_leituras_test.dart`
- [X] T060 [P] [US1] `app/lib/data/fila_leituras_local.dart`: caixa Hive `fila_leituras`, que descarta itens com mais de 7 dias e remove só o que o servidor confirmou

### Serviços

- [X] T061 [US1] `app/lib/orbit/leitor_orbit.dart`: busca BLE filtrada pelo UUID (`flutter_reactive_ble`) que emite `DeteccaoOrbit(orbitId, rssi, instante)`
- [X] T062 [US1] `app/lib/orbit/leituras_service.dart`:
  - junta cada detecção com a última posição de no máximo 60 s, deduplica, enfileira e faz o bip;
  - envia lotes de até 100 a cada 15 s e sinal de vida a cada 5 min;
  - mantém o contador da sessão;
  - dependências injetáveis, com teste em `app/test/leituras_service_test.dart`
- [X] T063 [US2] `app/lib/orbit/orbit_virtual.dart`: anúncio BLE (`flutter_ble_peripheral`), com UUID no pacote principal e Service Data na resposta de varredura; só Android
- [X] T064 [US1] `app/lib/orbit/ponto_transmissao.dart`: controlador da função, com estados desativada/ativa/suspensa, preferências em Hive (`preferencias`), consentimento no backend, permissões e início/parada junto com a sessão
- [X] T065 [US1] `app/lib/services/background_service.dart`: foreground service compartilhado por motivo (canal de voz e/ou ponto de transmissão), com os tipos de serviço certos para cada um
- [X] T066 [US1] `app/lib/services/localizacao_service.dart`: modo leitura (`PUT /posicao-atual`) para quem não tem empresa, usado na lista de canais

### Plataformas e telas

- [X] T067 [US1] Permissões Android (`BLUETOOTH_SCAN`, `BLUETOOTH_CONNECT`, `BLUETOOTH_ADVERTISE`, legadas até a API 30, `FOREGROUND_SERVICE_CONNECTED_DEVICE` e tipo `connectedDevice`) e iOS (`NSBluetoothAlwaysUsageDescription`, `UIBackgroundModes` com `bluetooth-central`)
- [X] T068 [US1] Cartão "Ponto de transmissão" na lista de canais, com explicação, consentimento, estado e leituras da sessão; ajustes de bip e vibração em "Minha conta"
- [X] T069 [US2] `app/lib/features/configuracoes/orbit_virtual_page.dart`: ID do Orbit, aviso permanente enquanto anuncia; só para `admin_ktm`/`demo`
- [ ] T070 Rodar `flutter analyze`, `flutter test`, `go test ./...` e compilar o app (feito: testes verdes e build iOS de dispositivo; pendente: APK, porque esta máquina não tem Android SDK, e o build de simulador, que falha no framework Opus)

**Checkpoint**: aceite em aparelhos reais (Bluetooth não funciona no simulador) e confirmação do usuário (Princípio II).

## Etapa 4 — Protótipo de UX (US6)

**Critério de aceite**: aprovação explícita do usuário (Princípio II).

- [X] T071 [US6] Protótipo navegável com 13 telas de celular (390×844), publicado como design: [Rádio PX KTM — protótipo de UX](https://claude.ai/artifact/RKYtV7SLZ8kK67ZDpZVqiX). Linhas:
  - primeiro acesso: 3 telas de apresentação, cadastro e permissões explicadas;
  - dia a dia: aba Canais, canal ativo, aba Explorar, criar canal e convite;
  - contribuição e conta: aba Contribuição, aba Perfil e Orbit virtual
- [X] T072 [US6] Aprovação do protótipo pelo usuário (aprovado em 2026-11-22, sem ajustes)

## Etapa 5 — Nova navegação e experiência (US4, US6)

**Objetivo**: o app segue o protótipo aprovado: apresentação inicial, permissões explicadas, 4 abas e o novo canal ativo.

**Critério de aceite**: teste com 3 pessoas que nunca usaram o app (SC-004/SC-005), **pendente**; `flutter test` verde (feito, exceto o `codec_voz_test`, que já falhava sem a biblioteca Opus no Mac).

**Fora desta etapa**: a aba Explorar e o "+" de criar canal mostram o estado "em breve", porque os canais da comunidade só chegam na Etapa 6.

- [X] T073 [US6] Tema em `app/lib/core/theme.dart`: claro e escuro (segue o sistema, FR-039), azul para a rádio, âmbar para a rede Orbit, e temas de `NavigationBar`, cartões e botões com alvos de no mínimo 48 px
- [X] T074 [US6] `app/lib/core/sessao_app.dart`: estado da sessão (logado, apresentação vista, permissões vistas) persistido em Hive, mais a função pura `decidirRota`, testada
- [X] T075 [US6] `app/lib/core/router.dart`: `go_router` com `StatefulShellRoute` de 4 abas (`/canais`, `/explorar`, `/contribuicao`, `/perfil`), rotas públicas (`/boas-vindas`, `/entrar`, `/cadastro`, `/recuperar-senha`, `/sobre`), `/permissoes`, `/canal/:id` e `/orbit-virtual` em tela cheia; substitui `routes.dart`
- [X] T076 [US4] `app/lib/features/onboarding/boas_vindas_page.dart`: 3 telas que podem ser puladas (FR-035)
- [X] T077 [US4] `app/lib/features/onboarding/permissoes_page.dart`: microfone, localização, Bluetooth e notificações, cada uma com explicação e botão próprio (FR-036/037)
- [X] T078 [US6] `app/lib/features/shell/casca_page.dart`: `NavigationBar` com as 4 abas (FR-038)
- [X] T079 [US6] Aba Canais (`app/lib/features/canais/canais_page.dart`, substitui `lista_canais_page.dart`) com o cartão compacto do ponto de transmissão
- [X] T080 [US6] Aba Explorar com o estado "em breve" (`app/lib/features/explorar/explorar_page.dart`)
- [X] T081 [US7] Aba Contribuição (`app/lib/features/contribuicao/contribuicao_page.dart`): `GET /me/contribuicao`, liga/desliga e bip
- [X] T082 [US6] Aba Perfil (`app/lib/features/perfil/perfil_page.dart`, substitui `conta_page.dart`): editar nome e apelido (`PATCH /me`), Orbit virtual, documentos, sair, excluir
- [X] T083 [US6] Novo visual de `canal_ativo_page.dart`: botão de falar grande e proporcional à tela, quem fala em destaque, fila, Silenciar e Sair sempre visíveis (FR-019); lógica de voz sem mudança
- [X] T084 Testes: `decidirRota`, apresentação → cadastro, navegação entre abas e `widget_test` atualizado; `flutter analyze`, `flutter test` e build iOS


## Etapa 6 — Canais da comunidade (US5)

**Objetivo**: qualquer pessoa cria canal público ou privado, encontra canais na aba Explorar, entra por código ou QR e marca favoritos. O criador convida, remove participantes e apaga o canal.

**Critério de aceite**: criar um canal privado e um segundo usuário entrar nele pelo QR em menos de 1 min (SC-006); testes de `PodeEntrar` 🔒 verdes.

**Ajuste no data-model**: a exclusão de canal usa a coluna `ativo`, que já existe em `canal`, em vez de criar `apagado_em`. O código de convite é guardado sem hífen (8 caracteres) e mostrado como `XXXX-XXXX`.

### Backend

- [X] T085 `backend/migrations/0006_canais_comunidade.sql`:
  - em `canal`: `empresa_id` aceita nulo; colunas novas `criador_id`, `visibilidade` e `codigo_convite`; criador preenchido nos canais antigos;
  - tabelas `acesso_canal` e `favorito_canal`;
  - índice `pg_trgm` no nome;
  - motivo de saída `removido`
- [X] T086 🔒 `backend/internal/canal/acesso.go`: função pura `PodeEntrar` (banido → apagado → sem permissão → fora da área → cheio), com teste em `acesso_test.go`
- [X] T087 `backend/internal/canal/convite.go`: gerar código sem caracteres ambíguos e normalizar o que o usuário digita, com teste
- [X] T088 `repositorio.go`/`service.go`:
  - `Criar` para canal da comunidade;
  - `ListarParaUsuario` (empresa ∪ criados ∪ convites ∪ favoritos, sem banidos);
  - `Explorar` (busca e proximidade);
  - `EntrarPorConvite`, `NovoConvite`, `Apagar`, `Remover` (banir), `Favoritar`, `Participantes`;
  - `ObterParaEntrada` passa a usar `PodeEntrar`
- [X] T089 `backend/internal/transporte/canais_handler.go` + rotas conforme [contracts/rest-api.md](./contracts/rest-api.md):
  - `GET /canais` e `GET /canais/explorar`, `GET /canais/{id}`;
  - `POST /canais` aberto a todos;
  - `PATCH`/`DELETE /canais/{id}` pelo criador ou admin;
  - `POST /canais/{id}/convite` e `POST /convites/{codigo}`;
  - `GET /canais/{id}/participantes` e `DELETE /canais/{id}/participantes/{usuarioId}`;
  - `PUT /canais/{id}/favorito`
- [X] T090 Testes de integração: `service_test.go` (fluxo do convite, banido não volta, privado não aparece na busca, apagar desconecta) e testes de contrato

### App

- [X] T091 [P] `app/lib/services/canais_api.dart`: cliente das rotas acima
- [X] T092 Aba Explorar real: busca, "Perto de mim" / "Todos os públicos", entrar, favoritar, e "Entrar com código ou QR"
- [X] T093 `app/lib/features/canais/criar_canal_page.dart` (público/privado, limite, área) e botão "+" na aba Canais
- [X] T094 `app/lib/features/canais/convite_page.dart`: QR (`qr_flutter`), código, copiar, compartilhar e gerar novo
- [X] T095 Canal ativo: menu do criador (convidar, participantes e remover, apagar), favoritar e avisos de removido ou apagado
- [X] T096 Deep link `radiopx://app/convite/<código>`: rota `/convite/:codigo`, `intent-filter` no Android e `CFBundleURLTypes` no iOS
- [X] T097 Testes: normalização do código e telas de Explorar e Criar canal; `flutter analyze`, `flutter test` e build iOS

**Checkpoint**: parar e pedir o aceite do usuário (Princípio II).

## Etapa 7 — Contribuição e privacidade (US7)

**Objetivo**: fechar os pontos de privacidade em aberto e conferir o SC-008 tela a tela. A aba Contribuição, os termos, a privacidade e o liga/desliga do ponto de transmissão já entraram na Etapa 5.

**Critério de aceite**: [checklists/privacidade.md](./checklists/privacidade.md) conferido tela a tela; testes verdes.

- [X] T098 Retenção automática (`backend/internal/retencao/`): apaga leituras Orbit e histórico de posições com mais de 90 dias e códigos de recuperação vencidos há mais de 1 dia, uma vez ao subir e depois a cada 24 h, com teste
- [X] T099 [US1] Revogar o consentimento descarta as leituras que ainda estão na fila do aparelho; `403 sem_consentimento` no envio também descarta a fila e suspende a função (o consentimento pode ter sido revogado em outro aparelho), com teste em `leituras_service_test.dart`
- [X] T100 Teste de privacidade das rotas do painel (`/rede/orbits`, histórico): nenhuma resposta contém id, nome, apelido ou e-mail de quem leu
- [X] T101 Atualizar `backend/painel/termos.html` e `privacidade.html`: canais da comunidade, convites e remoção, favoritos, retenção de 90 dias e fila local de leituras
- [X] T102 [`checklists/privacidade.md`](./checklists/privacidade.md): conferência do SC-008 e do Princípio VII tela a tela (app e painel), com a evidência de cada item e os riscos que ficam


## Etapa 8 — Preparação da demo

**Critério de aceite**: roteiro completo executado sem falhas 2 vezes seguidas (feito pelo usuário com os celulares).

- [X] T103 Seed final (`backend/cmd/seed/demo.go`, dados em `internal/demo`): canais públicos "Rádio KTM · Demo" (sem área) e "Trecho da demo" (com área), `-centro=lat,lng` para montar os pórticos em volta do local da apresentação e a flag `-zerar`, que devolve a demo ao estado inicial (apaga as leituras dos Orbits da demo e recoloca o caminhão 1 visto há 3 h no pórtico)
- [X] T104 Simulador de trajeto (`backend/cmd/simular/`): entra com a conta `demo` e envia leituras do caminhão andando pela BR-116, de dois pontos de transmissão. É o plano B ao vivo se o Bluetooth falhar e serve também para gravar o vídeo
- [X] T105 [`roteiro-demo.md`](./roteiro-demo.md): checklist da véspera e do dia, roteiro com o que falar e mostrar, o que fazer se cada passo falhar, como gravar o vídeo de plano B, protocolo de medição de bateria (SC-007) e do teste com 3 pessoas (SC-004/SC-005)
- [X] T106 Verificar a imagem Docker do backend, o seed `-demo`/`-zerar` e o simulador contra um banco limpo
- [ ] T107 Ensaiar o roteiro 2 vezes seguidas, gravar o vídeo e medir a bateria (usuário, com celulares reais)
