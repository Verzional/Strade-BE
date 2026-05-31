FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY main.go ./

RUN go mod init api-gateway && go build -o gateway-binary main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/gateway-binary .

COPY routes ./routes

EXPOSE 8080

CMD ["./gateway-binary"]