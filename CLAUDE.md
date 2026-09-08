# radio-px-ktm Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-09-08

## Active Technologies

- Go (backend + painel), Dart/Flutter (app móvel) + Backend — `pgx` (Postgres), `chi` (roteamento HTTP), `coder/websocket` (WebSocket), `golang-jwt` (JWT). App — `riverpod` (estado), `dio` (HTTP), `web_socket_channel`, `geolocator` (localização), `flutter_foreground_task` (segundo plano Android), `flutter_sound` (gravação/reprodução Opus em streaming). Painel — templates HTML nativos do Go + Leaflet (mapa). (001-radio-px-digital)

## Project Structure

```text
backend/    # API REST + hub WebSocket + painel web (Go) — ver specs/001-radio-px-digital/plan.md
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
- Identificadores de domínio, comentários e nomes de tabelas/campos em **português**, usando os termos fixados em `.specify/memory/constitution.md` (motorista, canal, empresa, geocerca, transmissão, fila, participação) — coerência com a defesa do TCC.
- Backend: SQL escrito à mão via `pgx`, sem ORM (Princípio V da constituição).
- **Áudio nunca é persistido** em disco, banco, log ou cache, em nenhuma camada (Princípio I) — ver `specs/001-radio-px-digital/spec.md` FR-008.

## Recent Changes

- 001-radio-px-digital: Added Go (backend + painel), Dart/Flutter (app móvel) + Backend — `pgx` (Postgres), `chi` (roteamento HTTP), `coder/websocket` (WebSocket), `golang-jwt` (JWT). App — `riverpod` (estado), `dio` (HTTP), `web_socket_channel`, `geolocator` (localização), `flutter_foreground_task` (segundo plano Android), `flutter_sound` (gravação/reprodução Opus em streaming). Painel — templates HTML nativos do Go + Leaflet (mapa).

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
