all: build run

build:
	GO111MODULE=on go build -o dist/pk-ingress-controller cmd/cmd.go
run:
	dist/pk-ingress-controller
