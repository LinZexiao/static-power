build:
	go build -o bin/$(BINARY_NAME) -v

test:
	go test -v ./...
clean:
	rm -rf bin/*
run:
	go run main.go
