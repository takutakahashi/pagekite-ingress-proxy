package types

import (
	"bytes"
	"fmt"
	"log"

	"github.com/leekchan/gtf"
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

func (pkc *PageKiteConfig) GenerateConfig(pki PageKiteIngress) string {
	tmpl, err := gtf.New("pagekite.rc.tmpl").ParseFiles("src/template/pagekite.rc.tmpl")
	if err != nil {
		log.Println(err)
		return ""
	}
	var buf bytes.Buffer
	type pkset struct {
		I PageKiteIngress
		C PageKiteConfig
	}
	err = tmpl.Execute(&buf, pkset{I: pki, C: *pkc})
	if err != nil {
		log.Println(err)
		return ""
	}
	return buf.String()
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
	delete(pki.Resources, ingress.GetUID())
	fmt.Println("delete:", len(pki.Resources))
}

func NewPageKiteIngress() PageKiteIngress {
	return PageKiteIngress{Resources: make(map[types.UID]*extv1beta1.Ingress)}
}
func NewPageKiteConfig() PageKiteConfig {
	return PageKiteConfig{}
}
