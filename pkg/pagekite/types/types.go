package types

type PageKiteConfig struct {
	Name   string
	Secret string
	Load   func()
}

type PageKiteIngress struct {
	Ingress string
}

func (pki *PageKiteIngress) Add(obj interface{}) {

}

func (pki *PageKiteIngress) Delete(obj interface{}) {

}

func (pki *PageKiteIngress) Update(old, new interface{}) {

}
