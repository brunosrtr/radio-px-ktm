FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -o /out/seed ./cmd/seed \
 && CGO_ENABLED=0 go build -o /out/simular ./cmd/simular

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/api /api
# Seed e simulador da demonstração (003-rede-orbit, roteiro-demo.md):
#   docker compose exec backend /seed -demo
#   docker compose exec backend /simular
COPY --from=build /out/seed /seed
COPY --from=build /out/simular /simular
ENTRYPOINT ["/api"]
