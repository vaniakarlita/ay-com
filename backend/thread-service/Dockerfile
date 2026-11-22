FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/thread-service ./cmd/server

FROM alpine:3.20

WORKDIR /app

RUN adduser -D -u 1000 appuser
USER appuser

COPY --from=builder /app/thread-service .

EXPOSE 50052

CMD ["./thread-service"]
