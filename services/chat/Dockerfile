FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o chat-binary main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/chat-binary .

EXPOSE 8080

CMD ["./chat-binary"]
