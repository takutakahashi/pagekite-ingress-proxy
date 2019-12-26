package pagekite

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/takutakahashi/pagekite-ingress-controller/pkg/pagekite/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	ccorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

type PageKite struct {
	Config types.PageKiteConfig
	Client ccorev1.ServiceInterface
	Stop   chan struct{}
	Reload chan struct{}
}

func NewPageKite() PageKite {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	namespace := *flag.String("namespace", "", "(optional) specify target namespace to watch resource")
	kitename := *flag.String("kitename", os.Getenv("PAGEKITE_NAME"), "kitename")
	kitesecret := *flag.String("kitesecret", os.Getenv("PAGEKITE_SECRET"), "kitesecret")
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
		Config: types.PageKiteConfig{Name: kitename, Secret: kitesecret},
		Client: clientset.CoreV1().Services(namespace),
		Stop:   make(chan struct{}),
		Reload: make(chan struct{}),
	}
	return pk
}

func (pk *PageKite) Start() error {
	pk.initProcess()
	pk.startObserver()
	return nil
}

func (pk *PageKite) initProcess() {
	svcList, err := pk.Client.List(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, svc := range svcList.Items {
		// TODO: detect ingress controller svc
		fmt.Println(svc.Name)
		pk.Config.ControllerService = svc
	}
	pk.generateConfig()
	pk.reloadProcess()
}

func (pk *PageKite) generateConfig() bool {
	config := pk.Config.GenerateConfig()
	fmt.Println(config)
	hasDiff := config != pk.Config.Cache
	pk.Config.Cache = config
	return hasDiff

}

func (pk *PageKite) startObserver() error {
	svcStreamWatcher, err := pk.Client.Watch(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	for {
		select {
		case event := <-svcStreamWatcher.ResultChan():
			svc := event.Object.(*v1.Service)
			pk.update(event.Type, svc)
		}
	}
}

func (pk *PageKite) update(eventType watch.EventType, svc *v1.Service) {
	switch eventType {
	case watch.Added:
		pk.Config.ControllerService = *svc
	case watch.Modified:
		pk.Config.ControllerService = *svc
	}
	hasDiff := pk.generateConfig()
	if hasDiff {
		pk.reloadProcess()
	}
}

func (pk *PageKite) reloadProcess() {
	fmt.Println("TODO: reload")
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
