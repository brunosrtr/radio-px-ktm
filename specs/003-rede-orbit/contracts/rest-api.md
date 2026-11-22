# Contrato REST — Rede Orbit (`003-rede-orbit`)

Complementa e substitui partes do [contrato REST da 001](../../001-radio-px-digital/contracts/rest-api.md).
Erros seguem o mesmo formato: `{"codigo": "...", "mensagem": "..."}`.

**Rotas públicas**: `POST /auth/cadastro`, `POST /auth/login`, `POST /auth/recuperar-senha`, `POST /auth/redefinir-senha`, `GET /termos`, `GET /privacidade`. Todas as outras exigem `Authorization: Bearer <jwt>`.

**JWT**: `usuario_id`, `papel` (`usuario` | `motorista` | `admin` | `admin_ktm` | `demo`) e `empresa_id` (vazio quando não há vínculo).

## Contas

| Método | Rota | Papel | Descrição |
|---|---|---|---|
| POST | `/auth/cadastro` | público | Cria conta do público (FR-023) |
| POST | `/auth/login` | público | `identificador` = e-mail ou login (contas de empresa) |
| POST | `/auth/recuperar-senha` | público | Envia código por e-mail (FR-025) |
| POST | `/auth/redefinir-senha` | público | Troca a senha com o código |
| GET | `/me` | todos | Dados do usuário + consentimentos vigentes |
| PATCH | `/me` | todos | Altera nome e apelido |
| DELETE | `/me` | todos | Exclui a conta (FR-026) |
| PUT | `/me/consentimentos/{tipo}` | todos | `{"aceito": true\|false}` (FR-028) |

### `POST /auth/cadastro`

```json
// Request
{ "nome": "Maria Souza", "apelido": "Maria da Serra", "email": "maria@exemplo.com",
  "senha": "********", "aceitou_termos_versao": "2026-11" }

// 201 Created
{ "token": "<jwt>", "usuario": { "id": "...", "apelido": "Maria da Serra", "papel": "usuario" } }

// 409 Conflict
{ "codigo": "email_em_uso", "mensagem": "este e-mail já tem conta" }
// 422 — dados_invalidos (senha < 8, apelido fora de 2–24, e-mail inválido, termos não aceitos)
// 429 — muitas_tentativas
```

### `POST /auth/recuperar-senha` / `POST /auth/redefinir-senha`

```json
{ "email": "maria@exemplo.com" }
// 202 Accepted — sempre, exista ou não a conta (não revela e-mails cadastrados)
// 503 — recuperacao_indisponivel (SMTP não configurado)

{ "email": "maria@exemplo.com", "codigo": "482913", "nova_senha": "********" }
// 204 No Content | 400 codigo_invalido (errado, expirado, usado ou 5 tentativas)
```

## Canais

| Método | Rota | Papel | Descrição |
|---|---|---|---|
| GET | `/canais` | todos | Canais acessíveis (meus, favoritos, liberados por convite, de empresa) |
| GET | `/canais/explorar?q=&lat=&lng=` | todos | Canais públicos por nome e/ou próximos (FR-030) |
| POST | `/canais` | todos | Cria canal da comunidade (FR-029) |
| PATCH | `/canais/{id}` | criador / admin da empresa | Edita |
| DELETE | `/canais/{id}` | criador / admin da empresa | Apaga (desconecta participantes) |
| POST | `/canais/{id}/convite` | criador | Gera novo código (invalida o anterior) |
| POST | `/convites/{codigo}` | todos | Entra por convite → `acesso_canal` liberado |
| DELETE | `/canais/{id}/participantes/{usuarioId}` | criador | Remove e bane (FR-031) |
| PUT | `/canais/{id}/favorito` | todos | `{"favorito": true\|false}` (FR-032) |
| PUT | `/canais/{id}/preferencia` | todos | Silenciar (001, sem mudança) |

As rotas `/canais/{id}/empresas` da 001 continuam para canais de empresa.

### `POST /canais`

```json
// Request
{ "nome": "Serra do Cafezal", "descricao": "Trecho BR-116 km 340–370",
  "limite_participantes": 20, "visibilidade": "publico",
  "geocerca": { "latitude": -24.27, "longitude": -47.93, "raio_metros": 15000 } }

// 201 Created
{ "id": "...", "nome": "Serra do Cafezal", "visibilidade": "publico",
  "codigo_convite": null, "sou_criador": true, ... }
```

Erros de entrada no canal (no `GET /canais/{id}` ou na entrada via WS): `banido`, `sem_permissao`, `fora_geocerca`, `canal_cheio`, `nao_encontrado` (ver data-model §3).

## Rede Orbit

| Método | Rota | Papel | Descrição |
|---|---|---|---|
| POST | `/leituras` | todos (com consentimento) | Lote de leituras (FR-004) |
| POST | `/ponto/sinal` | todos (com consentimento) | Sinal de vida do ponto de transmissão (FR-018) |
| GET | `/me/contribuicao` | todos | Contadores do usuário (FR-040) |
| PUT | `/posicao-atual` | todos (o app usa para contas sem empresa) | Um ponto, sem histórico (FR-034); ignora ponto mais antigo que o atual |
| GET | `/rede/orbits` | admin_ktm | Última posição de cada Orbit |
| GET | `/rede/orbits/{id}/leituras?de=&ate=` | admin_ktm | Histórico de leituras (FR-019) |
| GET/POST/DELETE | `/rede/pontos-fixos[/{id}]` | admin_ktm | Pontos fixos (FR-020) |
| POST | `/rede/pontos-fixos/{id}/simular` | admin_ktm | `{"orbit_id": "..."}`: gera uma leitura de origem `fixo` (demonstração) |
| GET | `/ws/rede` | admin_ktm | Eventos ao vivo ([painel-rede-ws.md](./painel-rede-ws.md)) |

### `POST /leituras`

```json
// Request (1–100 leituras)
{ "leituras": [
  { "orbit_id": "ORBIT-0A1B2C", "latitude": -23.5505, "longitude": -46.6333,
    "precisao_metros": 8.5, "rssi": -67, "detectado_em": "2026-11-21T14:03:22Z" }
] }

// 200 OK — o app remove da fila local todas as leituras enviadas
{ "aceitas": 1, "duplicadas": 0 }

// 403 sem_consentimento | 422 lote_invalido (vazio, > 100, orbit_id fora do formato,
//     detectado_em no futuro (> agora + 2 min) ou mais antigo que 7 dias)
```

O servidor **não** devolve nenhum dado do Orbit ao app (FR-011).

### `GET /me/contribuicao`

```json
{ "hoje": 3, "semana": 12, "total": 87, "ponto_ativo": true }
```
