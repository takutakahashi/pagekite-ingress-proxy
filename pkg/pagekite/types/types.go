package types

import (
	"bytes"
	"log"

	"github.com/leekchan/gtf"
	v1 "k8s.io/api/core/v1"
)

type PageKiteConfig struct {
	Name              string
	Secret            string
	Cache             string
	ControllerService v1.Service
}

func (pkc *PageKiteConfig) GenerateConfig() string {
	tmpl, err := gtf.New("pagekite.rc.tmpl").ParseFiles("src/template/pagekite.rc.tmpl")
	if err != nil {
		log.Println(err)
		return ""
	}
	var buf bytes.Buffer
	type pkset struct {
		C PageKiteConfig
		S v1.Service
	}
	err = tmpl.Execute(&buf, pkset{C: *pkc, S: pkc.ControllerService})
	if err != nil {
		log.Println(err)
		return ""
	}
	return buf.String()
}
