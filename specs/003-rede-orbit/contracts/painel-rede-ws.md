# Contrato WebSocket do painel Rede Orbit — `003-rede-orbit`

`GET /ws/rede`, restrito a `admin_ktm` (FR-021). Mensagens servidor → painel em JSON, no mesmo
envelope da 001: `{"tipo": "...", "dados": {...}}`. O painel não envia mensagens além do ping/pong
do protocolo.

Nenhum evento contém `usuario_id`, nome, apelido, e-mail ou telefone (FR-022, Princípio VII).

| `tipo` | Quando | `dados` |
|---|---|---|
| `inicial` | logo após conectar | `{ "orbits": [OrbitResumo], "pontos_fixos": [PontoFixo], "pontos_ativos": n }` |
| `leitura` | cada leitura nova aceita (não duplicada) | `{ "orbit_id", "origem": "celular"\|"fixo", "ponto_id", "latitude", "longitude", "rssi", "detectado_em" }` |
| `orbit_atualizado` | quando a última posição de um Orbit muda | `OrbitResumo` |
| `pontos_ativos` | a cada 30 s | `{ "total": n }` |

```json
// OrbitResumo
{ "id": "ORBIT-0A1B2C", "virtual": true, "rotulo": "Caminhão demo 1",
  "latitude": -23.5505, "longitude": -46.6333, "ultima_leitura_em": "2026-11-21T14:03:22Z",
  "leituras_24h": 42, "pontos_distintos_24h": 3 }

// PontoFixo
{ "id": "...", "nome": "Pórtico km 210", "latitude": -23.4, "longitude": -46.5,
  "raio_metros": 300, "simulado": true }
```

Um painel lento é desconectado em vez de atrasar a ingestão. O `Broadcaster` usa canais com buffer e
descarta o assinante que estiver cheio, o mesmo comportamento do painel de posições da 001.
