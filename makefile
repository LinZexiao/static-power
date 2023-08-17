build:
	go build -o bin/$(BINARY_NAME) -v

test:
	go test -v ./...
clean:
	rm -rf bin/*
run:
	go run main.go


list-miner:
	./bin/static-power update-peer --node=ws://192.168.200.18:3453/rpc/v1 --token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiZGVmYXVsdExvY2FsVG9rZW4iLCJwZXJtIjoiYWRtaW4iLCJleHQiOiIifQ.mTBWEuFTilRWUicmNVlJMyabPrhyBamBBTYPgA17iPs

	# ./bin/static-power update-peer --node=ws://192.168.200.132:3453/rpc/v1 --token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiYmFpeXVfdmVudXNfdGVzdF9mb3JjZXNlYWxlciIsInBlcm0iOiJ3cml0ZSIsImV4dCI6IiJ9.OfwrhnK-qasTd3iLM50BL1b3vYgIBz5_NRVcA-FsaKw

list-agent: 
	./bin/static-power 
