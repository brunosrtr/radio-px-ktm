# Contrato WebSocket

Feature: Rádio PX Digital v1 (`001-radio-px-digital`)

Endpoint: `GET /ws`, com o JWT no parâmetro de query ou no header
`Authorization`. A conexão é recusada antes do upgrade se o token for
inválido (FR-020/RNF11).

Mensagens de controle trafegam como frames de **texto** JSON no formato
`{"tipo": "...", "dados": {...}}`. O **áudio** trafega como frames
**binários** na mesma conexão, associado pelo servidor à transmissão ativa
daquela conexão (não há identificador de transmissão no frame binário em si —
uma conexão só tem uma transmissão ativa por vez).

## Cliente → Servidor

| Tipo | Dados | Efeito |
|---|---|---|
| `entrar_canal` | `canal_id` | Valida autorização de empresa, geocerca (se ativa) e limite de participantes; registra `participacao_canal` |
| `sair_canal` | `canal_id` | Saída manual; grava `motivo_saida = 'manual'` |
| `solicitar_slot` | `canal_id` | Pede permissão para transmitir (checa `len(fila) < 10`) |
| `finalizar_transmissao` | `transmissao_id` | Sinaliza fim da gravação |
| `ping` | — | Mantém a conexão viva |

Após receber `slot_concedido`, o cliente envia frames binários de áudio
(chunks Opus, conforme gerados pela gravação — ver research.md §7) na mesma
conexão, até enviar `finalizar_transmissao` ou atingir os 90 segundos
(cortado pelo servidor — ver research.md §3).

## Servidor → Cliente

| Tipo | Dados | Quando |
|---|---|---|
| `canal_entrado` | `canal_id`, `participantes`, `tamanho_fila` | Entrada aceita |
| `slot_concedido` | `transmissao_id`, `duracao_maxima_ms` | Havia espaço na fila (`duracao_maxima_ms` = 90000) |
| `slot_negado` | `motivo` | `fila_cheia`, `sem_permissao`, `fora_da_area` |
| `inicio_reproducao` | `transmissao_id`, `remetente_nome` | Antes dos frames de áudio (FR-021) |
| `fim_reproducao` | `transmissao_id` | Buffer descartado no servidor (FR-008) |
| `removido_canal` | `canal_id`, `motivo` | `geocerca`, `limite`, `canal_desativado` (FR-017) |
| `estado_canal` | `participantes`, `tamanho_fila` | Mudança relevante no canal (entrada/saída, fila muda de tamanho) |
| `erro` | `codigo`, `mensagem` | Falhas de validação |

## Sequência típica — transmissão bem-sucedida

```text
Cliente A                      Servidor                       Cliente B
   |--- entrar_canal --------------->|                              |
   |<-- canal_entrado ----------------|                              |
   |                                  |<--- entrar_canal ------------|
   |                                  |--- canal_entrado ----------->|
   |--- solicitar_slot -------------->|                              |
   |<-- slot_concedido ---------------|                              |
   |--- [frames binários Opus] ------>|                              |
   |                                  |--- inicio_reproducao ------->|
   |                                  |--- [frames binários] ------->|
   |--- finalizar_transmissao ------->|                              |
   |                                  |--- fim_reproducao ----------->|
```

## Regras de validação obrigatórias no servidor

1. `solicitar_slot`: se `len(fila) >= 10`, responder `slot_negado` com
   `motivo: "fila_cheia"` e **não** criar transmissão (FR-006/FR-007).
2. `entrar_canal`: se `count(participacao_canal ativa) >= limite_participantes`,
   responder `slot_negado`-equivalente para entrada (reaproveitar `erro` com
   `codigo: "limite_atingido"`, ou `slot_negado` adaptado — decisão de
   nomenclatura fica para a implementação, mantendo a semântica: entrada
   recusada, motivo explícito) (FR-023).
3. `entrar_canal` em canal com `geocerca_ativa = true`: calcular distância
   Haversine entre a última `posicao_atual` do motorista e o centro do canal;
   se fora do `raio_metros`, responder recusa com `motivo: "fora_da_area"`
   (FR-014/FR-015).
4. Transmissão ativa que ultrapasse 90 segundos de frames binários recebidos:
   servidor encerra a transmissão automaticamente, como se
   `finalizar_transmissao` tivesse sido recebido (FR-004).
5. Ao remover um motorista de um canal por geocerca (checagem periódica ou a
   cada atualização de posição), enviar `removido_canal` com
   `motivo: "geocerca"` e gravar `participacao_canal.motivo_saida = 'geocerca'`
   (FR-016/FR-017).
6. Ao concluir a reprodução de uma transmissão para todos os membros não
   silenciados, zerar o buffer de chunks em memória antes de processar o
   próximo item da fila (FR-008/FR-009).
