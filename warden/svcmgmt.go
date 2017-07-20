package warden

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	api_v1 "k8s.io/client-go/pkg/api/v1"
	ext_v1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetClientset() (*kubernetes.Clientset, error) {
	var clientset *kubernetes.Clientset
	var err error

	clientset, err = getClientsetInternal()
	if err == nil {
		return clientset, nil
	}
	clientset, err = getClientsetExternal()
	if err == nil {
		return clientset, nil
	}
	return nil, fmt.Errorf("[ERROR] No credentials available")

}

// For retrieving credentials outside of a Kubernetes cluster
func getClientsetExternal() (*kubernetes.Clientset, error) {
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")

	// Use the current context from the kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

// For retrieving credentials inside a Kubernetes cluster
func getClientsetInternal() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

func FindIngForFqdn(f string, c *kubernetes.Clientset) (ext_v1.Ingress, error) {
	//fmt.Printf("%s\n", f)
	opts := meta_v1.ListOptions{}
	ingresses, err := c.ExtensionsV1beta1().Ingresses("").List(opts)
	if err != nil {
		return ext_v1.Ingress{}, err
	}
	for _, i := range ingresses.Items {
		for _, r := range i.Spec.Rules {
			if r.Host == f {
				return i, nil
			}
		}
	}
	return ext_v1.Ingress{}, fmt.Errorf("[ERROR] No ingress found for domain %s\n", f)
}

func IsAutoManaged(s *api_v1.Service) bool {
	if _, ok := s.ObjectMeta.Annotations["ipautomanaged"]; ok {
		return true
	} else {
		return false
	}
}

func reconcileSourceRanges(c []string, n string, op string) ([]string, error) {
	if op == "add" {
		for _, v := range c {
			if v == n {
				return nil, fmt.Errorf("[INFO] IP address already has access.")
			}
		}
		c = append(c, n)
		return c, nil
	}
	if op == "remove" {
		for i, v := range c {
			if v == n {
				c[i] = c[0]
				return c[1:], nil
			}
		}
		return nil, fmt.Errorf("[INFO] IP address not found.")
	}
	return nil, fmt.Errorf("[ERROR] Unsupported operation %s\n", op)
}

func applySourceRangesToSpec(r []string, s *api_v1.Service) {
	s.Spec.LoadBalancerSourceRanges = r
}

func UpdateServiceSpec(iprange string, ns string, s *api_v1.Service, c *kubernetes.Clientset) error {
	ipranges, err := reconcileSourceRanges(s.Spec.LoadBalancerSourceRanges, iprange, "add")
	if err != nil {
		return err
	}
	applySourceRangesToSpec(ipranges, s)
	updateServiceAnnotation(iprange, s)
	_, err = c.CoreV1().Services(ns).Update(s)
	if err != nil {
		return err
	}
	return nil
}

func updateServiceAnnotation(iprange string, s *api_v1.Service) {
	now := time.Now()
	annotationKey := fmt.Sprintf("ipaddr.%s", iprange)
	annotationValue := fmt.Sprintf(now.Format("2006-01-02 15:04:05"))
	s.ObjectMeta.Annotations[annotationKey] = annotationValue
}

func removeServiceAnnotation(iprange string, s *api_v1.Service) {
	annotationKey := fmt.Sprintf("ipaddr.%s", iprange)
	delete(s.ObjectMeta.Annotations, annotationKey)
}

func IterateAnnotations(s *api_v1.Service, c *kubernetes.Clientset) error {
	now := time.Now()
	deadline := now.AddDate(0, 0, -2).Format("2006-01-02 15:04:05")
	fmt.Printf("The deadline is %s\n", deadline)
	for a, v := range s.ObjectMeta.Annotations {
		if strings.HasPrefix(a, "ipaddr") {
			if v < deadline {
				fmt.Printf("Time to remove this rule: %s\n", a)
				ip := strings.TrimPrefix(a, "ipaddr.")
				err := RemoveIpFromService(ip, s, c)
				if err != nil {
					return err
				}
			} else {
				fmt.Printf("Rule for %s has not expired yet\n", a)
			}
		}
	}
	return nil
}

func GetServiceList(c *kubernetes.Clientset) *api_v1.ServiceList {
	opts := meta_v1.ListOptions{}
	services, _ := c.CoreV1().Services("").List(opts)
	return services
}

func RemoveIpFromService(iprange string, s *api_v1.Service, c *kubernetes.Clientset) error {
	ns := s.ObjectMeta.Namespace
	ipranges, err := reconcileSourceRanges(s.Spec.LoadBalancerSourceRanges, iprange, "remove")
	if err != nil {
		return err
	}
	applySourceRangesToSpec(ipranges, s)
	removeServiceAnnotation(iprange, s)
	_, err = c.CoreV1().Services(ns).Update(s)
	if err != nil {
		return err
	}
	return nil
}

func ApplyRequestToCluster(data WhitelistRequest) error {
	var clientset *kubernetes.Clientset
	var err error

	clientset, err = GetClientset()
	if err != nil {
		return err
	}
	fmt.Printf("[INFO] Received ip address: %s\n", data.IpAddress)
	ing, err := FindIngForFqdn(data.Domain, clientset)
	if err != nil {
		return err
	}

	fmt.Printf("[INFO] Ingress name is: %s\n", ing.ObjectMeta.Name)
	fmt.Printf("[INFO] Service name is: %s\n", ing.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend.ServiceName)

	var service *api_v1.Service

	opts := meta_v1.GetOptions{}
	if ing.ObjectMeta.Annotations["kubernetes.io/ingress.class"] == "nginx" {
		// TODO: find the Nginx controller service dynamically
		service, _ = clientset.CoreV1().Services("default").Get("ingress-nginx", opts)
	} else {
		return fmt.Errorf("[ERROR] Only the Nginx ingress controller is supported.")
	}
	fmt.Printf("[INFO] The service to modify: %s\n", service.ObjectMeta.Name)
	if !IsAutoManaged(service) {
		return fmt.Errorf("[ERROR] The service is not auto-managed.\n")
	}
	namespace := service.ObjectMeta.Namespace
	err = UpdateServiceSpec(data.IpAddress, namespace, service, clientset)
	if err != nil {
		return err
	}
	fmt.Printf("[INFO] Successfully applied %s to the service for %s\n", data.IpAddress, data.Domain)
	return nil
}
