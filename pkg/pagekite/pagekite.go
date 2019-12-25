package pagekite

import (
	"time"

	"github.com/takutakahashi/pagekite-ingress-controller/pkg/pagekite/types"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type PageKite struct {
	Config  types.PageKiteConfig
	Ingress types.PageKiteIngress
	Stop    chan struct{}
	Reload  chan struct{}
}

func NewPageKite() PageKite {
	pk := PageKite{
		Config:  types.PageKiteConfig{},
		Ingress: types.PageKiteIngress{},
		Stop:    make(chan struct{}),
		Reload:  make(chan struct{}),
	}
	return pk
}

func (pk *PageKite) StartObserver() error {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	watchlist := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"ingresses",
		v1.NamespaceAll,
		fields.Everything(),
	)

	_, controller := cache.NewInformer(
		watchlist,
		&v1beta1.Ingress{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pk.Ingress.Add(obj)
				pk.Reload <- struct{}{}
			},
			DeleteFunc: func(obj interface{}) {
				pk.Ingress.Delete(obj)
				pk.Reload <- struct{}{}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				pk.Ingress.Update(oldObj, newObj)
				pk.Reload <- struct{}{}
			},
		},
	)
	go controller.Run(pk.Stop)
	for {
		time.Sleep(time.Second)
	}
}
