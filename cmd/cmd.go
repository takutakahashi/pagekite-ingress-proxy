package main

import (
	"github.com/takutakahashi/pagekite-ingress-controller/pkg/pagekite"
)

func main() {
	pk := pagekite.NewPageKite()
	go pk.StartServer()
	go pk.StartObserver()
}
