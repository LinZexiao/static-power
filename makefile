build:
	go build -o bin/$(BINARY_NAME) -v
	
test:
	go test -v ./...
clean:
	rm -rf bin/*
run:
	go run main.go


list-miner:
	./bin/static-power -list-miners --node=ws://192.168.200.18:3453/rpc/v1 --token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiZGVmYXVsdExvY2FsVG9rZW4iLCJwZXJtIjoiYWRtaW4iLCJleHQiOiIifQ.rwwZMSLKuvWiv9A8oHQLJdAdgXl5NIFXRlucRetfkn4

list-agent: 
	./bin/static-power 
