FROM golang:alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o server ./cmd/server/

FROM alpine:latest

RUN apk add --no-cache libc6-compat ca-certificates \
    && addgroup -S skillhub \
    && adduser -S -G skillhub skillhub

WORKDIR /app

COPY --from=builder /app/server .

RUN mkdir -p uploads thumbnails avatars \
    && chown -R skillhub:skillhub /app

USER skillhub

EXPOSE 8080

CMD ["./server"]
