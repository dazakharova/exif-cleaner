# Build stage
FROM golang:1.24-alpine AS builder
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags='-s' -o /out/stripper ./cmd/api

# Runtime stage
FROM scratch AS final
COPY --from=builder --chown=1000:1000 /out/stripper /stripper
USER 1000:1000
EXPOSE 8080
ENTRYPOINT ["/stripper"]
