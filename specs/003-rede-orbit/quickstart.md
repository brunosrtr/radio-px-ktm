# Quickstart — Rede Orbit (`003-rede-orbit`)

Como subir o ambiente e rodar a **demonstração do bip** para a KTM. Os passos marcados com
_(Etapa N)_ só funcionam depois que a etapa correspondente do [plan.md](./plan.md) for concluída.

## 1. Subir o backend

```bash
cp .env.example .env          # preencher JWT_SECRET, PONTO_ID_SEGREDO e SMTP_* (Mailpit já vem configurado)
docker compose up             # Postgres + backend + Mailpit (http://localhost:8025)
cd backend && go run ./cmd/seed -demo   # (Etapa 2) cria admin_ktm, conta demo, 3 Orbits virtuais e 4 pontos fixos simulados
```

## 2. Instalar o app em três celulares

| Celular | Sistema | Conta | Papel na demo |
|---|---|---|---|
| A | **Android** | `demo` | Orbit virtual (fica "no caminhão") |
| B | Android ou iPhone | conta nova criada na hora (mostra o cadastro aberto) | Ponto de transmissão |
| C | Android ou iPhone | qualquer conta do público | Ponto de transmissão + rádio |

```bash
cd app && flutter run -d <id-do-celular>
```

Todos na mesma rede do notebook (a descoberta local da 001 continua) ou com `--dart-define=API_BASE_URL=...`.

## 3. Painel

Abra `http://<ip-do-notebook>:8080/painel/rede.html` e entre com a conta `admin_ktm` do seed _(Etapa 2)_.

## 4. Roteiro da apresentação (cerca de 5 minutos)

1. **O problema**: no painel, ative "pontos fixos". Mostre o caminhão `ORBIT-0A1B2C` com a última leitura antiga e os pontos fixos espaçados. Use "simular passagem por ponto fixo" para mostrar como é hoje: só há posição quando o caminhão passa por um pórtico.
2. **Cadastro aberto**: no celular B, crie uma conta na hora, passe pela apresentação inicial e aceite ser ponto de transmissão _(Etapa 1/5)_.
3. **O bip**: no celular A, ative **Modo Orbit virtual** _(Etapa 3)_. Aproxime B de A: B vibra e toca o bip, e em até 5 s o painel pisca a leitura e move o caminhão (SC-001).
4. **A rede**: afaste B, aproxime C. O painel mostra a nova leitura vinda de outro ponto e o contador de pontos ativos sobe para 2.
5. **Segundo plano**: bloqueie a tela de C e aproxime de A de novo. A leitura chega, só que com mais atraso (FR-006).
6. **Sem internet**: coloque B em modo avião, aproxime de A, tire do modo avião. A leitura chega com o horário real (SC-003).
7. **A rádio**: B e C entram num canal e conversam. É o motivo pelo qual as pessoas mantêm o app aberto.
8. **Privacidade**: mostre que B não vê dados do caminhão e que o painel não mostra quem fez a leitura (SC-008).

**Plano B**: vídeo gravado do roteiro completo _(Etapa 8)_, caso o Bluetooth ou a rede do local falhem.

## 5. Testes

```bash
cd backend && go test ./...   # inclui ingestão de leituras, atualização condicional do Orbit, PodeEntrar
cd app && flutter test        # inclui DeduplicadorLeituras, parser do sinal Orbit, fila de leituras
```
