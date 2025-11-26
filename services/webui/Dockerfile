# Build stage
FROM golang:1.24-alpine AS builder
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags='-s' -o /out/webui ./cmd/api

# Runtime stage
FROM alpine:3.21 AS final
WORKDIR /app
COPY --from=builder --chown=1000:1000 /out/webui /app/webui
USER 1000:1000
COPY public /app/public

ENV PORT=3000
EXPOSE 3000

ENTRYPOINT ["/app/webui"]
