# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.24.2 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN go build -o /app/backend-bin main.go
RUN go build -o /app/worker-bin ./worker

# Final image
FROM alpine:3.19
RUN apk add --no-cache ffmpeg
WORKDIR /app
COPY --from=builder /app/backend-bin /app/backend
COPY --from=builder /app/worker-bin /app/worker
RUN chmod +x /app/backend /app/worker
RUN mkdir -p /app/uploads
COPY uploads /app/uploads
EXPOSE 3000
ENTRYPOINT ["/app/backend"]