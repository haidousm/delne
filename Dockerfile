FROM golang:1.21.5
WORKDIR /src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN make web/build

RUN curl -JLO "https://dl.filippo.io/mkcert/latest?for=linux/amd64"
RUN chmod +x mkcert-v*-linux-amd64
RUN cp mkcert-v*-linux-amd64 /usr/local/bin/mkcert

EXPOSE 80 443

CMD ["./bin/web"]
