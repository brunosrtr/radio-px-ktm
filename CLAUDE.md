# radio-px-ktm Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-11-21

## Active Technologies
- Acréscimos da 003-rede-orbit — App: `flutter_reactive_ble` (leitura BLE do sinal Orbit; não usar `flutter_blue_plus`, que exige licença comercial), `flutter_ble_peripheral` (Orbit virtual), `go_router` (navegação/deep link), `qr_flutter` (QR de convite). Backend: `net/smtp` (recuperação de senha), migrações `0004`–`0006`; Hive ganha a caixa `fila_leituras`. Formato do sinal Orbit só em `app/lib/orbit/orbit_sinal.dart`; integração KTM só via `orbit.Integracao`. (003-rede-orbit)

- Go (backend + painel), Dart/Flutter (app móvel) + Backend — `pgx` (Postgres), `chi` (roteamento HTTP), `coder/websocket` (WebSocket), `golang-jwt` (JWT). App — `riverpod` (estado), `dio` (HTTP), `web_socket_channel`, `geolocator` (localização), `flutter_foreground_task` (segundo plano Android), `flutter_sound` (gravação/reprodução Opus em streaming). Painel — templates HTML nativos do Go + Leaflet (mapa). (001-radio-px-digital)

## Project Structure

```text
backend/    # API REST + hub WebSocket + painel web (Go) — ver specs/001-radio-px-digital/plan.md
            # escopo atual: specs/003-rede-orbit/spec.md (rede aberta + leituras Orbit)
app/        # app Flutter (Android/iOS)
```

## Commands

```bash
docker compose up             # sobe Postgres + backend local
cd backend && go test ./...   # testes do backend (unitários + integração com Postgres real)
cd app && flutter test        # testes do app
```

## Code Style

- Go (backend + painel), Dart/Flutter (app móvel): seguir as convenções padrão de cada linguagem.
- Identificadores de domínio, comentários e nomes de tabelas/campos em **português**, usando os termos fixados em `.specify/memory/constitution.md` (usuário, motorista, canal, convite, empresa, geocerca, transmissão, fila, participação, Orbit, Orbit virtual, leitura Orbit, ponto de transmissão, ponto fixo de leitura) — coerência com a defesa do TCC.
- Backend: SQL escrito à mão via `pgx`, sem ORM (Princípio V da constituição).
- **Áudio nunca é persistido** em disco, banco, log ou cache, em nenhuma camada (Princípio I) — ver `specs/001-radio-px-digital/spec.md` FR-008.

## Recent Changes

- 003-rede-orbit: escopo redefinido — app aberto ao público (cadastro próprio, canais da comunidade), celular como ponto de transmissão de leituras Orbit e modo Orbit virtual para demonstração. Constituição v2.0.0 (Princípio VII: privacidade em mão dupla).

- 001-radio-px-digital: Added Go (backend + painel), Dart/Flutter (app móvel) + Backend — `pgx` (Postgres), `chi` (roteamento HTTP), `coder/websocket` (WebSocket), `golang-jwt` (JWT). App — `riverpod` (estado), `dio` (HTTP), `web_socket_channel`, `geolocator` (localização), `flutter_foreground_task` (segundo plano Android), `flutter_sound` (gravação/reprodução Opus em streaming). Painel — templates HTML nativos do Go + Leaflet (mapa).

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
