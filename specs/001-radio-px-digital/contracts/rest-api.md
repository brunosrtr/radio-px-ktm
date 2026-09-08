# Contrato REST

Feature: Rádio PX Digital v1 (`001-radio-px-digital`)

Todas as rotas, exceto `/auth/login`, exigem `Authorization: Bearer <jwt>`.
O JWT carrega `usuario_id`, `empresa_id` e `papel` (`admin` ou `motorista`).
Uma requisição sem token válido recebe `401` antes de qualquer lógica de
negócio ser executada (FR-020/RNF11).

| Método | Rota | Papel | Descrição |
|---|---|---|---|
| POST | `/auth/login` | público | Autentica e devolve o token |
| GET | `/me` | ambos | Dados do usuário autenticado |
| GET | `/canais` | motorista | Canais disponíveis, já filtrados por autorização de empresa e por geocerca |
| POST | `/canais` | admin | Cria canal |
| PATCH | `/canais/{id}` | admin | Edita canal |
| POST | `/canais/{id}/empresas` | admin | Libera o canal para outra empresa |
| DELETE | `/canais/{id}/empresas/{empresaId}` | admin | Revoga a liberação |
| PUT | `/canais/{id}/preferencia` | motorista | Silencia ou reativa o canal |
| POST | `/posicoes` | motorista | Recebe um lote de posições |
| GET | `/empresas/{id}/posicoes-atuais` | admin | Última posição de cada motorista da empresa |
| GET | `/motoristas/{id}/trajeto?de=&ate=` | admin | Histórico de posições no período |

## Exemplos

### `POST /auth/login`

```json
// Request
{ "login": "motorista01", "senha": "..." }

// 200 OK
{ "token": "<jwt>", "usuario": { "id": "...", "nome": "...", "papel": "motorista" } }

// 401 Unauthorized (credenciais inválidas)
{ "codigo": "credenciais_invalidas", "mensagem": "login ou senha incorretos" }
```

### `GET /canais` (motorista)

Retorna apenas canais autorizados para a empresa do motorista (privados da
própria empresa + compartilhados liberados para ela — FR-010, FR-012, FR-029)
e, para canais com geocerca ativa, apenas os que o motorista está atualmente
dentro do raio (FR-014, FR-015). A posição usada para o filtro de geocerca é a
última `posicao_atual` conhecida do motorista.

```json
// 200 OK
{
  "canais": [
    {
      "id": "...",
      "nome": "Rota BR-386 Norte",
      "tipo_acesso": "privado",
      "limite_participantes": 10,
      "participantes_atual": 4,
      "geocerca_ativa": true,
      "silenciado": false
    }
  ]
}
```

### `POST /canais` (admin)

```json
// Request
{
  "nome": "Rota BR-386 Norte",
  "descricao": "Canal do trecho de Marau a Passo Fundo",
  "limite_participantes": 10,
  "tipo_acesso": "privado",
  "geocerca_ativa": true,
  "centro_latitude": -28.4497,
  "centro_longitude": -52.2003,
  "raio_metros": 5000
}

// 201 Created -> corpo do canal criado

// 422 Unprocessable Entity (limite fora de {5,10,15,20}, ou geocerca ativa sem centro/raio)
{ "codigo": "canal_invalido", "mensagem": "limite_participantes deve ser 5, 10, 15 ou 20" }
```

### `PUT /canais/{id}/preferencia` (motorista)

```json
// Request
{ "silenciado": true }

// 200 OK -> preferência atualizada
```

### `POST /posicoes` (motorista)

Insere todos os pontos do lote em `posicao` numa única operação e atualiza
`posicao_atual` apenas com o ponto de `capturado_em` mais recente do lote, e
somente se for mais novo que o já gravado (ver research.md §8, data-model.md).

```json
// Request
{
  "pontos": [
    {
      "latitude": -28.4497,
      "longitude": -52.2003,
      "precisao_metros": 8.5,
      "velocidade_kmh": 82.4,
      "capturado_em": "2026-09-08T14:31:02Z"
    }
  ]
}

// 202 Accepted
{ "pontos_recebidos": 1 }

// 400 Bad Request (lote vazio ou ponto sem capturado_em)
{ "codigo": "lote_invalido", "mensagem": "pontos[0].capturado_em é obrigatório" }
```

### `GET /empresas/{id}/posicoes-atuais` (admin)

Só retorna dados se `{id}` for a própria `empresa_id` do token do admin
(FR-026 restrito à própria empresa — nunca a de outra empresa).

```json
// 200 OK
{
  "motoristas": [
    {
      "usuario_id": "...",
      "nome": "João",
      "latitude": -28.4497,
      "longitude": -52.2003,
      "velocidade_kmh": 82.4,
      "capturado_em": "2026-09-08T14:31:02Z"
    }
  ]
}

// 403 Forbidden (tentativa de consultar outra empresa)
{ "codigo": "sem_permissao", "mensagem": "acesso restrito à própria empresa" }
```

### `GET /motoristas/{id}/trajeto?de=&ate=` (admin)

`{id}` deve pertencer à mesma `empresa_id` do admin autenticado.

```json
// 200 OK
{
  "trajeto": [
    { "latitude": -28.4497, "longitude": -52.2003, "capturado_em": "2026-09-08T14:31:02Z" }
  ]
}

// 200 OK, sem dados no período (FR-027, cenário "sem erro")
{ "trajeto": [] }
```

## Códigos de erro comuns

| `codigo` | Quando |
|---|---|
| `credenciais_invalidas` | login/senha incorretos |
| `token_invalido` | JWT ausente, expirado ou malformado (401) |
| `sem_permissao` | ação fora do papel do usuário ou fora da própria empresa (403) |
| `canal_invalido` | corpo de criação/edição de canal não passa nas validações (422) |
| `lote_invalido` | corpo de `/posicoes` malformado (400) |
| `nao_encontrado` | recurso inexistente ou fora do alcance do usuário (404) |
