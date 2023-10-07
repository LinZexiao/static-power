git=$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))

ldflags=-X=main.CurrentCommit=+git.$(git)
ifneq ($(strip $(LDFLAGS)),)
	ldflags+=-extldflags=$(LDFLAGS)
endif

GOFLAGS+=-ldflags="$(ldflags)"

build:
	echo $(ldflags)
	go build -o bin/$(BINARY_NAME) -v $(GOFLAGS)

test:
	go test -v ./...
clean:
	rm -rf bin/*
run:
	go run main.go
