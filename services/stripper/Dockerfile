# Build stage
FROM golang:1.24-alpine AS builder
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/stripper ./cmd/api

# Runtime stage
FROM alpine:3.21 AS final
COPY --from=builder /out/stripper /stripper
EXPOSE 8000
ENTRYPOINT ["/stripper"]
