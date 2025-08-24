# build
FROM golang:1.24.6-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /build/anomaly-consumer ./cmd/main.go

# runtime
FROM alpine:latest
WORKDIR /app
RUN apk --no-cache add ca-certificates && adduser -D -g '' appuser
COPY --from=builder /build/anomaly-consumer /app/anomaly-consumer
USER appuser
ENTRYPOINT ["/app/anomaly-consumer"]
