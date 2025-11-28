tidy:
	go mod tidy

verify:
	go mod verify

vet:
	go vet ./...

staticcheck:
	staticcheck ./...

test:
	go test -vet=off ./...

check: tidy verify vet staticcheck test

build:
	go build -o bin/app .
