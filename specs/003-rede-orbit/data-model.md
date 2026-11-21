# Modelo de dados — Rede Orbit (`003-rede-orbit`)

Alterações sobre o modelo da 001 ([data-model.md](../001-radio-px-digital/data-model.md)), aplicadas pela migração
`backend/migrations/0004_rede_orbit.sql`: idempotente, sem apagar dados (Princípio V). Nenhuma tabela guarda áudio (Princípio I).

## 1. Usuário e contas

### `usuario` (alterada)

| Campo | Mudança | Regra |
|---|---|---|
| `empresa_id` | passa a aceitar `null` | `null` = usuário do público (FR-033) |
| `email` | **novo**, `citext` | único quando não nulo; obrigatório para `papel = 'usuario'` |
| `apelido` | **novo**, `text` | 2–24 caracteres; nome exibido no rádio (FR-027). Contas antigas: preenchido com `primeiro_nome` ou `nome` |
| `login` | passa a aceitar `null` | continua único; só contas de empresa usam |
| `papel` | check ampliado | `usuario`, `motorista`, `admin`, `admin_ktm`, `demo` |
| `excluido_em` | **novo**, `timestamptz` | exclusão de conta (FR-026): dados pessoais anonimizados, `ativo = false` |

Restrições novas: `check (email is not null or login is not null)`; `check (papel in ('motorista','admin') = (empresa_id is not null))`, ou seja, papéis de empresa exigem empresa e os demais não têm.

### `recuperacao_senha` (nova)

| Campo | Tipo | Regra |
|---|---|---|
| `id` | uuid pk | |
| `usuario_id` | uuid fk → usuario | |
| `codigo_hash` | text | bcrypt do código de 6 dígitos |
| `expira_em` | timestamptz | criação + 15 min |
| `tentativas` | int | código invalidado após 5 tentativas erradas |
| `usado_em` | timestamptz null | uso único |

### `consentimento` (nova)

| Campo | Tipo | Regra |
|---|---|---|
| `usuario_id` | uuid fk | |
| `tipo` | text | `termos` \| `ponto_transmissao` |
| `versao` | text | versão do texto aceito |
| `aceito_em` | timestamptz | |
| `revogado_em` | timestamptz null | `ponto_transmissao` pode ser revogado a qualquer momento (FR-008/028) |

pk `(usuario_id, tipo, versao)`. Consentimento vigente = o registro mais recente do tipo com `revogado_em is null`.

## 2. Posição (comportamento alterado, sem mudança de esquema)

- `posicao` (histórico): só recebe pontos de usuários com `empresa_id` não nulo (FR-034). O serviço recusa o lote de quem não tem empresa com `403 sem_historico`. O app nem envia, ver research §5.
- `posicao_atual`: continua para **todos** os usuários, porque é a base do filtro de geocerca e dos canais próximos. Para quem não tem empresa, é atualizada por `PUT /posicao-atual` (um ponto, sem histórico). É apagada na exclusão da conta.

## 3. Canais

### `canal` (alterada)

| Campo | Mudança | Regra |
|---|---|---|
| `empresa_id` | passa a aceitar `null` | `null` = canal da comunidade |
| `criador_id` | **novo**, uuid fk → usuario | obrigatório em canais novos; canais antigos: o primeiro `admin` da empresa |
| `visibilidade` | **novo** | `publico` \| `privado`; padrão `publico`. Canais antigos: `privado` (mantém o comportamento da 001) |
| `codigo_convite` | **novo**, varchar(8) | único; gerado para canais privados; renovável (FR-031) |
| `apagado_em` | **novo**, timestamptz | exclusão lógica (preserva participações para auditoria) |

`limite_participantes` continua com o check `(5,10,15,20)`. Índice de busca: `lower(nome)` com `pg_trgm` para busca por nome (FR-030).

### `acesso_canal` (nova)

Quem entrou por convite num canal privado ou foi banido dele.

| Campo | Tipo | Regra |
|---|---|---|
| `canal_id` | uuid fk | |
| `usuario_id` | uuid fk | |
| `situacao` | text | `liberado` \| `banido` |
| `atualizado_em` | timestamptz | |

pk `(canal_id, usuario_id)`. Banir = `situacao = 'banido'` mais desconexão imediata do hub (US5 cenário 4).

### `favorito_canal` (nova)

pk `(usuario_id, canal_id)`, `criado_em` (FR-032).

### Regra de entrada (função pura `canal.PodeEntrar`, testes obrigatórios)

```
banido                                   → recusa (banido)
canal apagado                            → recusa (nao_encontrado)
não (público ou criador ou liberado
     ou canal de empresa autorizado)     → recusa (sem_permissao)
geocerca ativa e fora do raio            → recusa (fora_geocerca)
participantes ≥ limite                   → recusa (canal_cheio)
caso contrário                           → aceita
```

## 4. Rede Orbit

### `orbit` (nova)

| Campo | Tipo | Regra |
|---|---|---|
| `id` | varchar(12) pk | ex.: `ORBIT-0A1B2C` |
| `virtual` | boolean | `true` para Orbits do modo demonstração |
| `rotulo` | text null | placa ou apelido fictício (demo); dado sensível, visível só no painel |
| `ultima_latitude`, `ultima_longitude` | numeric(9,6) null | última posição conhecida |
| `ultima_leitura_em` | timestamptz null | atualizado **só** se a leitura nova for mais recente (FR-009) |
| `criado_em` | timestamptz | criado automaticamente na primeira leitura |

### `leitura_orbit` (nova)

| Campo | Tipo | Regra |
|---|---|---|
| `id` | bigserial pk | |
| `orbit_id` | varchar(12) fk → orbit | |
| `origem` | text | `celular` \| `fixo` |
| `ponto_id` | varchar(16) | opaco: `HMAC(segredo, usuario_id ‖ dia)` para celular; id do ponto fixo para `fixo`. **Nunca** `usuario_id` (Princípio VII) |
| `latitude`, `longitude` | numeric(9,6) | posição do ponto de transmissão |
| `precisao_metros` | numeric(6,2) null | |
| `rssi` | smallint null | intensidade do sinal (dBm) |
| `detectado_em` | timestamptz | horário real no celular; recusa se > agora + 2 min |
| `recebido_em` | timestamptz default now() | |

Índices: `(orbit_id, detectado_em desc)` para histórico (FR-019) e `(recebido_em)` para retenção de 90 dias e para os pontos ativos. Unicidade `(orbit_id, ponto_id, detectado_em)` com `on conflict do nothing`, para que reenviar um lote depois de falha de rede não duplique leituras.

### `ponto_fixo` (nova)

| Campo | Tipo | Regra |
|---|---|---|
| `id` | uuid pk | |
| `nome` | text | |
| `latitude`, `longitude` | numeric(9,6) | |
| `raio_metros` | int | 50–5000 |
| `simulado` | boolean | `true` na demonstração |

### `contribuicao_diaria` (nova)

Contadores de "Minha contribuição" (FR-040), separados de `leitura_orbit` para que esta nunca aponte para o usuário.

| Campo | Tipo | Regra |
|---|---|---|
| `usuario_id` | uuid fk | |
| `dia` | date | fuso `America/Sao_Paulo` |
| `leituras` | int | incrementado na ingestão do lote com as leituras **novas** (não duplicadas) |

pk `(usuario_id, dia)`. É apagada na exclusão da conta.

### `ponto_ativo` (em memória, não persistido)

Mapa `ponto_id → visto_em` no processo, alimentado por `POST /leituras` e pelo sinal de vida `POST /ponto/sinal` (a cada 5 min enquanto a função está ativa). O painel conta as entradas com `visto_em` nos últimos 15 min (FR-018). Não precisa sobreviver a reinício (Princípio VI).

## 5. App (armazenamento local)

| Caixa Hive | Conteúdo | Regra |
|---|---|---|
| `fila_posicoes` | sem mudança (001) | só usada por usuários com empresa |
| `fila_leituras` | `{orbitId, lat, lng, precisao, rssi, detectadoEm}` | descarta itens com mais de 7 dias; removidos só após `200` do servidor |
| `preferencias` | apresentação concluída, bip sonoro/vibração, função de ponto ativa | |

## 6. Transições de estado

**Função de ponto de transmissão** (app):

```
desativada ──consentir + permissões ok──▶ ativa ──sem Bluetooth/permissão──▶ suspensa
    ▲                                       │  ▲                                │
    └──────────── usuário desliga ──────────┘  └──── permissão recuperada ──────┘
```

**Orbit virtual** (app, papéis `admin_ktm`/`demo`): `parado → anunciando → parado` (botão ou fechar o app). Enquanto está `anunciando`, o leitor do mesmo aparelho ignora o próprio `orbitId`.
