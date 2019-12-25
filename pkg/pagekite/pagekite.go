package pagekite

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/takutakahashi/pagekite-ingress-controller/pkg/pagekite/types"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

type PageKite struct {
	Config     types.PageKiteConfig
	Ingress    types.PageKiteIngress
	KubeClient *kubernetes.Clientset
	Stop       chan struct{}
	Reload     chan struct{}
}

func NewPageKite() PageKite {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	//	config, err = rest.InClusterConfig()
	//	if err != nil {
	//		panic(err)
	//	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	pk := PageKite{
		Config:     types.NewPageKiteConfig(),
		Ingress:    types.NewPageKiteIngress(),
		Stop:       make(chan struct{}),
		Reload:     make(chan struct{}),
		KubeClient: clientset,
	}
	return pk
}

func (pk *PageKite) Start() error {
	pk.initProcess()
	pk.startObserver()
	return nil
}

func (pk *PageKite) initProcess() {

}

func (pk *PageKite) startObserver() error {
	ingressStreamWatcher, err := pk.KubeClient.ExtensionsV1beta1().Ingresses("").Watch(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	for {
		select {
		case event := <-ingressStreamWatcher.ResultChan():
			ingress := event.Object.(*extv1beta1.Ingress)
			pk.update(event.Type, ingress)
		}
	}
}

func (pk *PageKite) update(eventType watch.EventType, ingress *extv1beta1.Ingress) {
	switch eventType {
	case watch.Added:
		pk.Ingress.Add(ingress)
	case watch.Modified:
		pk.Ingress.Update(ingress)
	case watch.Deleted:
		pk.Ingress.Delete(ingress)
	}
	needed := pk.Config.GenerateConfig(pk.Ingress)
	if needed {
		pk.reloadProcess()
	}
}

func (pk *PageKite) reloadProcess() {
	fmt.Println("reload")
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
