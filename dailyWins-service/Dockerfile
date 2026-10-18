FROM golang:1.25-bookworm AS builder
RUN apt-get update && apt-get install -y --no-install-recommends protobuf-compiler \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
COPY go.mod go.sum Makefile ./
RUN go mod download && make init
COPY . .
# Generate from source so a clean Git checkout builds without local artifacts.
RUN make api && make config && make wire && go test ./... \
    && CGO_ENABLED=0 make build

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 10001 app \
    && mkdir -p /app /logs && chown app:app /app /logs
WORKDIR /app
COPY --from=builder /src/bin/dailyWins-service /app/dailyWins-service
COPY configs/config.yaml configs/config-prod.yaml /data/conf/
USER app
EXPOSE 10006 11006
ENTRYPOINT ["/app/dailyWins-service"]
CMD ["-conf", "/data/conf", "-env", "prod"]
