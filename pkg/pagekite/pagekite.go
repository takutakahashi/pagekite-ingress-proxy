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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ccorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
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
	namespace := *flag.String("namespace", os.Getenv("INGRESS_CONTROLLER_SERVICE"), "specify target namespace to watch resource")
	kitename := *flag.String("kitename", os.Getenv("PAGEKITE_NAME"), "kitename")
	kitesecret := *flag.String("kitesecret", os.Getenv("PAGEKITE_SECRET"), "kitesecret")
	controllerService := *flag.String("ingress-controller-service-name", os.Getenv("INGRESS_CONTROLLER_SERVICE"), "ingress svc")
	flag.Parse()

	// use the current context in kubeconfig
	config, err := rest.InClusterConfig()
	if config == nil {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			panic(err)
		}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	fmt.Println(namespace)
	pk := PageKite{
		Config: types.PageKiteConfig{
			Name:                  kitename,
			Secret:                kitesecret,
			ControllerServiceName: controllerService,
		},
		Client: clientset.CoreV1().Services(namespace),
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
	svcStreamWatcher, err := pk.Client.Watch(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	for {
		select {
		case event := <-svcStreamWatcher.ResultChan():
			svc := event.Object.(*v1.Service)
			if svc.Name == pk.Config.ControllerServiceName {
				pk.update(svc)
			}
		}
	}
}

func (pk *PageKite) update(svc *v1.Service) {
	pk.Config.ControllerService = *svc
	hasDiff := pk.generateConfig()
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
	out, err := cmd.Output()
	fmt.Println(out)
	pid := cmd.Process.Pid
	f, err := os.Create("/tmp/pagekite.pid")
	if err != nil {
		fmt.Printf("error creating file: %v", err)
		return
	}
	defer f.Close()
	_, err = f.WriteString(fmt.Sprintf("%d\n", pid))
	if err != nil {
		fmt.Printf("error writing string: %v", err)
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
