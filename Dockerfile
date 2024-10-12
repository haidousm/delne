FROM golang:1.21.5 AS builder
WORKDIR /src/app

RUN apt-get update && apt-get install -y make curl gcc

ENV CGO_ENABLED=1

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN make web/build

RUN curl -JLO "https://dl.filippo.io/mkcert/latest?for=linux/amd64" && \
    chmod +x mkcert-v*-linux-amd64 && \
    mv mkcert-v*-linux-amd64 /usr/local/bin/mkcert

FROM debian:bullseye-slim
WORKDIR /src/app

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /src/app/bin/web /src/app/
COPY --from=builder /usr/local/bin/mkcert /usr/local/bin/mkcert

EXPOSE 80 443

CMD ["./web"]

