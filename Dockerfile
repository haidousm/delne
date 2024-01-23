FROM golang:1.21.5
WORKDIR /src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN make build/web


EXPOSE 4000

CMD ["./bin/web"]