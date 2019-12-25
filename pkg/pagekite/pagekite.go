package pagekite

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/takutakahashi/pagekite-ingress-controller/pkg/pagekite/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
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
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func (pk *PageKite) StartObserver() error {
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

	serviceStreamWatcher, err := clientset.CoreV1().Services("").Watch(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	for {
		select {
		case event := <-serviceStreamWatcher.ResultChan():
			service := event.Object.(*v1.Service)

			for key, value := range service.Labels {
				fmt.Printf("Key, VAlue: %s %s\n", key, value)
			}
		}
	}
}
