package pagekite

import (
	"github.com/takutakahashi/pagekite-ingress-controller/pkg/pagekite/types"
)

type PageKite struct {
	Config  types.PageKiteConfig
	Ingress types.PageKiteIngress
}

func NewPageKite() PageKite {
	pk := PageKite{}
	return pk
}
func (pk *PageKite) StartServer() {

}

func (pk *PageKite) StartObserver() {

}
