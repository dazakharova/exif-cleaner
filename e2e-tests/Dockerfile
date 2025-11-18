# Build stage
FROM golang:1.24-alpine AS builder
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .

ENTRYPOINT ["go", "test", "./...", "-v"]