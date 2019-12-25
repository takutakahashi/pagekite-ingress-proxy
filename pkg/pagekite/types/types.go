package types

import (
	"fmt"

	extv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/types"
)

type PageKiteConfig struct {
	Name   string
	Secret string
	Cache  string
}

type PageKiteIngress struct {
	Resources map[types.UID]*extv1beta1.Ingress
}

func (pkc *PageKiteConfig) GenerateConfig(pki PageKiteIngress) bool {
	fmt.Println("gen")
	return true
}

func (pki *PageKiteIngress) Add(ingress *extv1beta1.Ingress) {
	pki.Resources[ingress.GetUID()] = ingress
	fmt.Println("add:", len(pki.Resources))
}
func (pki *PageKiteIngress) Update(ingress *extv1beta1.Ingress) {
	pki.Resources[ingress.GetUID()] = ingress
	fmt.Println("update:", len(pki.Resources))
}
func (pki *PageKiteIngress) Delete(ingress *extv1beta1.Ingress) {
	pki.Resources[ingress.GetUID()] = nil
	fmt.Println("delete:", len(pki.Resources))
}

func NewPageKiteIngress() PageKiteIngress {
	return PageKiteIngress{Resources: make(map[types.UID]*extv1beta1.Ingress)}
}
func NewPageKiteConfig() PageKiteConfig {
	return PageKiteConfig{}
}
