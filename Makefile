all: build

build:
	GO111MODULE=on go build -o dist/pk-ingress-controller cmd/cmd.go
