package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// List Services
	prompt()
	namespace := "default"
	servicesList, err := clientset.CoreV1().Services(apiv1.NamespaceDefault).List(metav1.ListOptions{})
	if err != nil {
		log.Printf("Get service from kubernetes cluster error:%v", err)
		return
	}
	for _, service := range servicesList.Items {
		log.Println("service:", service.GetName())
		if namespace == "default" && service.GetName() == "kubernetes" {
			continue
		}
		log.Println(
			"namespace", namespace,
			"serviceName:", service.GetName(),
			"serviceKind:", service.Kind,
			"serviceLabels:", service.GetLabels(),
			"servicePort:", service.Spec.Ports,
			"serviceSelector:", service.Spec.Selector,
		)

		// labels.Parser
		set := labels.Set(service.Spec.Selector)

		if pods, err := clientset.Core().Pods(namespace).List(metav1.ListOptions{LabelSelector: set.AsSelector().String()}); err != nil {
			log.Printf("List Pods of service[%s] error:%v", service.GetName(), err)
		} else {
			for _, v := range pods.Items {
				log.Println(v.GetName(), v.Spec.NodeName, v.Spec.Containers)
			}
		}
	}

	// List Deployments
	prompt()
	deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)
	fmt.Printf("Listing deployments in namespace %q:\n", apiv1.NamespaceDefault)
	deployList, err := deploymentsClient.List(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, deploy := range deployList.Items {
		fmt.Printf(" * %s (%d replicas)\n", deploy.GetName(), *deploy.Spec.Replicas)

		// labels.Parser
		set := labels.Set(deploy.Spec.Selector.MatchLabels)

		if pods, err := clientset.Core().Pods(namespace).List(metav1.ListOptions{LabelSelector: set.AsSelector().String()}); err != nil {
			log.Printf("List Pods of deploy[%s] error:%v", deploy.GetName(), err)
		} else {
			for _, v := range pods.Items {
				log.Println(v.GetName(), v.Spec.NodeName, v.Spec.Containers)
				for _, c := range v.Spec.Containers {
					if c.Resources.Requests == nil || c.Resources.Limits == nil {
						fmt.Printf(" * container: %s got empty resources requests %v limit %v\n",
							c.Name, c.Resources.Requests, c.Resources.Limits)
						deleteDeployment(deploy.GetName(), deploymentsClient)
						break
					} else {
						fmt.Printf(" * container: %s, resources requests %v limit %v\n",
							c.Name, c.Resources.Requests, c.Resources.Limits)
					}
				}
			}
		}
	}

	// List Pods
	prompt()
	podsClient := clientset.CoreV1().Pods(apiv1.NamespaceDefault)
	var podName string
	fmt.Printf("Listing pods in namespace %q:\n", apiv1.NamespaceDefault)
	podList, err := podsClient.List(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, p := range podList.Items {
		podName = p.Name
		fmt.Printf(" * pod: %s\n", p.Name)
		for _, c := range p.Spec.Containers {
			fmt.Printf(" * container: %s\n", c.Name)
		}
		break
	}

	// Delete Pod
	prompt()
	fmt.Printf("Deleting pod %s...\n", podName)
	deletePolicy := metav1.DeletePropagationForeground
	if err := podsClient.Delete(podName, &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		panic(err)
	}
	fmt.Println("Deleted pod.")
}

func prompt() {
	fmt.Printf("-> Press Return key to continue.")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		break
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	fmt.Println()
}

func deleteDeployment(name string, deploymentsClient v1.DeploymentInterface) {
	// Delete Deployment
	prompt()
	fmt.Println("Deleting deployment...")
	deletePolicy := metav1.DeletePropagationForeground
	if err := deploymentsClient.Delete(name, &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		panic(err)
	}
	fmt.Println("Deleted deployment.")
}

func int32Ptr(i int32) *int32 { return &i }
