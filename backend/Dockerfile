# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.24.2 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN go build -o backend main.go

# Final image
FROM alpine:3.19
RUN apk add --no-cache ffmpeg
WORKDIR /app
COPY --from=builder /app/backend /app/backend
RUN chmod +x /app/backend
RUN mkdir -p /app/uploads
COPY uploads /app/uploads
EXPOSE 3000
ENTRYPOINT ["/app/backend"]