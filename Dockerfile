FROM golang:1.21 AS builder

WORKDIR /usr/src/chatcord
COPY . .
RUN go mod tidy
RUN go build -o app .

FROM debian:latest

# Throws x509 cert error on discord.com if you don't install ca-certificates
RUN apt-get update && \
    apt-get install -y ca-certificates && \
    update-ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /chatcord
COPY --from=builder /usr/src/chatcord/app /chatcord/

CMD ["./app"]