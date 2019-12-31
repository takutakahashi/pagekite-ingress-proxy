package pagekite

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/takutakahashi/pagekite-ingress-controller/pkg/pagekite/types"
	v1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type PageKite struct {
	Config types.PageKiteConfig
	Client *kubernetes.Clientset
	Stop   chan struct{}
	Reload chan struct{}
}

func handle(err error) {
	if err != nil {
		panic(err)
	}
}

func NewPageKite() PageKite {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	namespace := *flag.String("namespace", os.Getenv("INGRESS_CONTROLLER_SERVICE"), "specify target namespace to watch resource")
	kitename := *flag.String("kitename", os.Getenv("PAGEKITE_NAME"), "kitename")
	kitesecret := *flag.String("kitesecret", os.Getenv("PAGEKITE_SECRET"), "kitesecret")
	controllerService := *flag.String("ingress-controller-service-name", os.Getenv("INGRESS_CONTROLLER_SERVICE"), "ingress svc")
	flag.Parse()

	config, err := rest.InClusterConfig()
	if config == nil {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		handle(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	handle(err)
	ings, err := clientset.ExtensionsV1beta1().Ingresses("").List(metav1.ListOptions{})
	handle(err)
	svc, err := clientset.CoreV1().Services(namespace).Get(controllerService, metav1.GetOptions{})
	handle(err)
	pk := PageKite{
		Config: types.PageKiteConfig{
			Name:   kitename,
			Secret: kitesecret,
			Resource: types.PageKiteResource{
				Ingresses:                ings.Items,
				IngressControllerService: *svc,
			},
		},
		Client: clientset,
		Stop:   make(chan struct{}),
		Reload: make(chan struct{}),
	}
	return pk
}

func (pk *PageKite) Start() error {
	pk.startObserver()
	return nil
}

func (pk *PageKite) generateConfig() bool {
	config := pk.Config.GenerateConfig()
	hasDiff := !bytes.Equal(config, pk.Config.Cache)
	homedir, err := os.UserHomeDir()
	if err != nil {
		homedir = "/tmp"
	}
	ioutil.WriteFile(homedir+"/.pagekite.rc", config, 0644)
	pk.Config.Cache = config
	return hasDiff

}

func (pk *PageKite) startObserver() error {
	go pk.watchIngress()
	go pk.watchService()
	<-pk.Stop
	return nil
}

func (pk *PageKite) watchService() {
	ns := pk.Config.Resource.IngressControllerService.Namespace
	streamWatcher, err := pk.Client.CoreV1().Services(ns).Watch(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	for {
		select {
		case event := <-streamWatcher.ResultChan():
			svc := event.Object.(*v1.Service)
			if svc.Name == pk.Config.Resource.IngressControllerService.Name {
				pk.update(svc, nil)
			}
		}
	}

}

func (pk *PageKite) watchIngress() {
	ns := ""
	ingressClient := pk.Client.ExtensionsV1beta1().Ingresses(ns)
	streamWatcher, err := ingressClient.Watch(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	for {
		select {
		case <-streamWatcher.ResultChan():
			res, err := ingressClient.List(metav1.ListOptions{})
			if err != nil {
				panic(err.Error())
			}
			pk.update(nil, res.Items)
		}
	}

}

func (pk *PageKite) update(svc *v1.Service, ingresses []extv1beta1.Ingress) {
	if svc != nil {
		pk.Config.Resource.IngressControllerService = *svc
	}
	if ingresses != nil {
		pk.Config.Resource.Ingresses = ingresses
	}
	hasDiff := pk.generateConfig()
	fmt.Println("diff? ", hasDiff)
	if hasDiff {
		pk.reloadProcess()
	}
}

func (pk *PageKite) reloadProcess() {
	buf, err := ioutil.ReadFile("/tmp/pagekite.pid")
	if err == nil {
		var pid int
		r := bytes.NewReader(buf)
		binary.Read(r, binary.LittleEndian, &pid)
		process, err := os.FindProcess(pid)
		if err != nil {
			fmt.Println(err)
		}
		process.Kill()
	}
	cmd := exec.Command("pagekite.py")
	_, err = cmd.Output()
	handle(err)
	pid := cmd.Process.Pid
	f, err := os.Create("/tmp/pagekite.pid")
	handle(err)
	defer f.Close()
	_, err = f.WriteString(fmt.Sprintf("%d\n", pid))
	handle(err)
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
