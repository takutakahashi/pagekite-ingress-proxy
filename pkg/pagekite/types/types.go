package types

type PageKiteConfig struct {
	Name   string
	Secret string
}

type PageKiteIngress struct {
	Ingress string
}

type PageKite struct {
	Config  PageKiteConfig
	Ingress PageKiteIngress
}

func NewPageKite() PageKite {
	pk := PageKite{}
	return pk
}
