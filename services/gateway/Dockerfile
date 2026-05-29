FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /gateway .

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /gateway /usr/local/bin/gateway
COPY routes ./routes

EXPOSE 8080

CMD ["gateway"]
