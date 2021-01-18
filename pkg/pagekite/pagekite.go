package pagekite

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"
	"sort"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/go-cmd/cmd"
	"github.com/prometheus/common/log"
	"github.com/takutakahashi/pagekite-ingress-controller/pkg/pagekite/types"
	v1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type PageKite struct {
	Config          types.PageKiteConfig
	Client          *kubernetes.Clientset
	HealthcheckPath string
	Stop            chan struct{}
	Reload          chan struct{}
}

func NewPageKite() PageKite {

	healthcheckPath := *flag.String("heealthcheck-path", "/healthz", "heal check path")
	namespace := *flag.String("namespace", os.Getenv("INGRESS_CONTROLLER_SERVICE_NAMESPACE"), "ingress svc namespace")
	kitename := *flag.String("kitename", os.Getenv("PAGEKITE_NAME"), "kitename")
	kitesecret := *flag.String("kitesecret", os.Getenv("PAGEKITE_SECRET"), "kitesecret")
	controllerService := *flag.String("ingress-controller-service-name", os.Getenv("INGRESS_CONTROLLER_SERVICE"), "ingress svc")
	flag.Parse()
	config := ctrl.GetConfigOrDie()
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error(err)
	}
	ings, err := clientset.NetworkingV1beta1().Ingresses("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Error(err)
	}
	svc, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), controllerService, metav1.GetOptions{})
	if err != nil {
		log.Error(err)
	}
	pk := PageKite{
		Config: types.PageKiteConfig{
			Name:   kitename,
			Secret: kitesecret,
			Resource: types.PageKiteResource{
				Ingresses:                ings.Items,
				IngressControllerService: *svc,
			},
		},
		Client:          clientset,
		HealthcheckPath: healthcheckPath,
		Stop:            make(chan struct{}),
		Reload:          make(chan struct{}),
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

func (pk *PageKite) healthcheck() {
	for {
		hcmd := exec.Command("bash", "-c", "netstat -anp |grep python")
		_, err := hcmd.Output()
		if err != nil {
			fmt.Println("heealthcheck failed. reload process.")
			pk.reloadProcess()
		}
		time.Sleep(5 * time.Second)
	}
}

func (pk *PageKite) health() bool {
	hcmd := exec.Command("bash", "-c", "netstat -anp |grep python")
	_, err := hcmd.Output()
	return err == nil

}

func (pk *PageKite) startObserver() error {
	go pk.watchIngress()
	// go pk.watchService()
	go pk.healthcheck()
	<-pk.Stop
	return nil
}

func (pk *PageKite) watchService() {
	ns := pk.Config.Resource.IngressControllerService.Namespace
	streamWatcher, err := pk.Client.CoreV1().Services(ns).Watch(context.TODO(), metav1.ListOptions{})
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
	ingressClient := pk.Client.NetworkingV1beta1().Ingresses(ns)
	streamWatcher, err := ingressClient.Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Error(err)
		return
	}
	for {
		select {
		case <-streamWatcher.ResultChan():
			res, err := ingressClient.List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				panic(err.Error())
			}
			pk.update(nil, res.Items)
		}
	}

}

func (pk *PageKite) update(svc *v1.Service, ingresses []networkingv1beta1.Ingress) {
	if svc != nil {
		pk.Config.Resource.IngressControllerService = *svc
	}
	if ingresses != nil {
		pk.Config.Resource.Ingresses = ingresses
	}
	sort.Slice(ingresses, func(i, j int) bool { return ingresses[i].Name < ingresses[j].Name })
	hasDiff := pk.generateConfig()
	fmt.Println("diff? ", hasDiff)
	if hasDiff {
		pk.reloadProcess()
	}
}

func (pk *PageKite) reloadProcess() {
	k := exec.Command("killall", "pagekite.py")
	k.Output()
	c := cmd.NewCmd("pagekite.py")
	c.Start()
	for {
		if pk.health() {
			break
		}
		time.Sleep(5 * time.Second)
	}
}
