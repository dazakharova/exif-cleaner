# Build stage
FROM golang:1.24-alpine AS builder
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o /e2e-test-runner main.go

# Runtime stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /e2e-test-runner /app/e2e-test-runner
COPY --from=builder /app/testdata /app/testdata
ENTRYPOINT ["/app/e2e-test-runner"]