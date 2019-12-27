package types

import (
	"bytes"
	"log"

	"github.com/leekchan/gtf"
	v1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
)

type PageKiteConfig struct {
	Name     string
	Secret   string
	Cache    []byte
	Resource PageKiteResource
}

type PageKiteResource struct {
	IngressControllerService v1.Service
	Ingresses                []extv1beta1.Ingress
}

func (pkc *PageKiteConfig) GenerateConfig() []byte {
	tmpl, err := gtf.New("pagekite.rc.tmpl").ParseFiles("src/template/pagekite.rc.tmpl")
	if err != nil {
		log.Println(err)
		return []byte{}
	}
	var buf bytes.Buffer
	type pkset struct {
		C PageKiteConfig
		S v1.Service
	}
	err = tmpl.Execute(&buf, pkset{C: *pkc, S: pkc.Resource.IngressControllerService})
	if err != nil {
		log.Println(err)
		return []byte{}
	}
	return buf.Bytes()
}
