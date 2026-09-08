# Fase 1 — Modelo de Dados

Feature: Rádio PX Digital v1 (`001-radio-px-digital`)

Regra central (Princípio I da constituição): **o PostgreSQL nunca armazena
áudio**. Nenhuma entidade abaixo representa uma mensagem de voz — a fila e os
buffers de transmissão existem apenas em memória no hub do backend (ver
research.md §2 e §4) e não têm contrapartida em tabela.

## Entidades

### Empresa
Organização cliente do serviço.

| Campo | Tipo | Regras |
|---|---|---|
| `id` | uuid (PK) | gerado |
| `razao_social` | text | obrigatório |
| `cnpj` | varchar(14) | obrigatório, único |
| `ativa` | boolean | default `true` |
| `criado_em` | timestamptz | default `now()` |

### Usuário
Pessoa autenticada, vinculada a uma empresa. Papel `admin` ou `motorista`
(FR-020, RF21).

| Campo | Tipo | Regras |
|---|---|---|
| `id` | uuid (PK) | gerado |
| `empresa_id` | uuid (FK → empresa) | obrigatório |
| `nome` | text | obrigatório — usado para identificar o remetente de cada transmissão (FR-021) |
| `login` | text | obrigatório, único |
| `senha_hash` | text | obrigatório, nunca em texto plano |
| `papel` | text | check `in ('admin', 'motorista')` |
| `ativo` | boolean | default `true` |
| `criado_em` | timestamptz | default `now()` |

Índice: `(empresa_id) where ativo` — consultas de listagem por empresa
(RNF12) são sempre filtradas por usuário ativo.

### Dispositivo
Registro do dispositivo móvel do usuário (plataforma, versão do app, token de
push para avisos como remoção por geocerca — FR-017).

| Campo | Tipo | Regras |
|---|---|---|
| `id` | uuid (PK) | gerado |
| `usuario_id` | uuid (FK → usuario) | obrigatório |
| `plataforma` | text | check `in ('android', 'ios')` |
| `push_token` | text | opcional |
| `versao_app` | text | opcional |
| `visto_em` | timestamptz | default `now()` |

### Canal
Espaço de comunicação por voz de uma empresa (FR-009 a FR-012, FR-022).

| Campo | Tipo | Regras |
|---|---|---|
| `id` | uuid (PK) | gerado |
| `empresa_id` | uuid (FK → empresa) | dono do canal |
| `nome` | text | obrigatório |
| `descricao` | text | opcional |
| `tipo_acesso` | text | check `in ('privado', 'compartilhado')`, default `'privado'` |
| `limite_participantes` | int | check `in (5, 10, 15, 20)`, default `10` — FR-022 |
| `geocerca_ativa` | boolean | default `false` |
| `centro_latitude` | numeric(9,6) | obrigatório se `geocerca_ativa` |
| `centro_longitude` | numeric(9,6) | obrigatório se `geocerca_ativa` |
| `raio_metros` | int | obrigatório se `geocerca_ativa` |
| `ativo` | boolean | default `true` |
| `criado_em` | timestamptz | default `now()` |

Constraint `geocerca_completa`: geocerca só pode estar ativa se centro e raio
estiverem preenchidos (evita canal com geocerca ativa mas sem parâmetros).

### Liberação de Canal para Empresa (Canal_Empresa)
Resolve FR-012/FR-013: quais empresas podem acessar um canal. Um canal
`privado` só tem a linha da empresa dona; um `compartilhado` ganha uma linha
por empresa parceira convidada.

| Campo | Tipo | Regras |
|---|---|---|
| `canal_id` | uuid (FK → canal, on delete cascade) | PK composta |
| `empresa_id` | uuid (FK → empresa) | PK composta |
| `liberado_em` | timestamptz | default `now()` |

### Participação em Canal
Histórico de entradas/saídas de um usuário em um canal (FR-016, FR-018,
FR-023).

| Campo | Tipo | Regras |
|---|---|---|
| `id` | uuid (PK) | gerado |
| `canal_id` | uuid (FK → canal) | obrigatório |
| `usuario_id` | uuid (FK → usuario) | obrigatório |
| `entrou_em` | timestamptz | default `now()` |
| `saiu_em` | timestamptz | nulo enquanto ativo |
| `motivo_saida` | text | check `in ('manual','geocerca','desconexao','encerramento')`, nulo enquanto ativo |

Índice único parcial `idx_participacao_ativa` em `(canal_id, usuario_id) where
saiu_em is null`: garante no banco que um usuário não tenha duas participações
ativas no mesmo canal, e `count(*) where saiu_em is null` por `canal_id`
alimenta a checagem de limite de participantes (FR-023).

### Preferência de Canal
Indica se um usuário silenciou um canal (FR-019), sem sair dele.

| Campo | Tipo | Regras |
|---|---|---|
| `usuario_id` | uuid (FK → usuario) | PK composta |
| `canal_id` | uuid (FK → canal, on delete cascade) | PK composta |
| `silenciado` | boolean | default `false` |

### Posição
Histórico bruto de localização (FR-024, FR-025). Cresce rápido — ver nota de
volume abaixo.

| Campo | Tipo | Regras |
|---|---|---|
| `id` | bigserial (PK) | gerado |
| `usuario_id` | uuid (FK → usuario) | obrigatório |
| `latitude` | numeric(9,6) | obrigatório |
| `longitude` | numeric(9,6) | obrigatório |
| `precisao_metros` | numeric(6,2) | opcional |
| `velocidade_kmh` | numeric(5,2) | opcional |
| `capturado_em` | timestamptz | obrigatório — horário real de captura no dispositivo |
| `recebido_em` | timestamptz | default `now()` — horário de chegada ao servidor |

`capturado_em` e `recebido_em` divergem propositalmente (podem estar até
horas distantes em trechos sem sinal — FR-028). Índice
`(usuario_id, capturado_em desc)` para consultas de trajeto por período
(FR-027).

**Nota de volume**: ~2.900 linhas/dia por motorista em movimento contínuo
(um ponto a cada ~30s). Retenção mínima de 90 dias (FR-025/RNF16).
Particionamento mensal é desejável mas opcional — só entra em consideração
depois que as etapas obrigatórias do plano estiverem concluídas (ver seção
"Como conduzir a execução" do briefing).

### Posição Atual
Última posição conhecida de cada motorista — é a única tabela consultada pelo
mapa do painel (FR-026); nunca se consulta `posicao` para montar o mapa.

| Campo | Tipo | Regras |
|---|---|---|
| `usuario_id` | uuid (PK, FK → usuario) | uma linha por motorista |
| `latitude` | numeric(9,6) | obrigatório |
| `longitude` | numeric(9,6) | obrigatório |
| `velocidade_kmh` | numeric(5,2) | opcional |
| `capturado_em` | timestamptz | obrigatório |
| `atualizado_em` | timestamptz | default `now()` |

Atualização condicional: só sobrescreve se o `capturado_em` recebido for mais
recente que o já gravado (ver research.md §8) — evita que um lote atrasado
regrida a posição exibida.

## Relacionamentos

```text
empresa 1──n usuario
empresa 1──n canal (dono)
empresa n──n canal (via canal_empresa, liberação para parceiras)
canal 1──n participacao_canal
usuario 1──n participacao_canal
usuario 1──n dispositivo
usuario 1──n preferencia_canal (por canal)
usuario 1──n posicao
usuario 1──1 posicao_atual
```

## DDL

O DDL completo (idempotente, com `create extension if not exists pgcrypto` e
todas as tabelas/índices/constraints acima) já está definido na seção 6 do
briefing original e deve ser versionado em `backend/migrations/` como a
primeira migração da Etapa 1 — não é reproduzido aqui para evitar duas fontes
de verdade; qualquer alteração de schema durante a implementação deve ser
feita via nova migração versionada, nunca editando uma migração já aplicada.

## Regras de negócio que exigem teste automatizado (Princípio IV)

- Fila do hub: FIFO, capacidade máxima 10, recusa com `fila_cheia` quando
  cheia, buffer zerado após reprodução.
- Limite de participantes: entrada bloqueada quando
  `count(saiu_em is null) >= limite_participantes`.
- Geocerca: função de distância (Haversine) e decisão de entrada/remoção a
  partir do raio configurado.
- Ingestão de lote de posições: inserção em lote em `posicao` +
  atualização condicional de `posicao_atual` apenas quando `capturado_em` for
  mais recente.
