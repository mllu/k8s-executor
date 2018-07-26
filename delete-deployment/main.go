package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/mllu/k8s-executor/config"
	appv1 "k8s.io/api/apps/v1"
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
	slackToken := flag.String("token", "", "slack token")
	slackChannel := flag.String("channel", "", "slack channel")
	flag.Parse()

	kbConf, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	kubecli, err := kubernetes.NewForConfig(kbConf)
	if err != nil {
		panic(err)
	}

	// to change the flags on the default logger
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	namespace := "default"
	// namespace = apiv1.NamespaceDefault

	// initialize argument configurations
	conf, err := config.New(namespace, *slackToken, *slackChannel)
	if err != nil {
		panic(err)
	}

	// initialize slack client
	slackCli, err := InitSlack(conf)
	if err != nil {
		panic(err)
	}

	// List Deployments
	prompt()
	deploymentsClient := kubecli.AppsV1().Deployments(namespace)
	log.Printf("Listing deployments in namespace %q:\n", namespace)
	deployList, err := deploymentsClient.List(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, deploy := range deployList.Items {

		pods, err := listPodsByDeployLabel(kubecli, namespace, deploy)
		if err != nil {
			log.Printf("List Pods of deploy[%s] error:%v", deploy.GetName(), err)
			continue
		}
		shouldDeleteDeployment := false
		for _, pod := range pods.Items {
			for _, c := range pod.Spec.Containers {
				if c.Resources.Requests == nil || c.Resources.Limits == nil {
					log.Printf(" * pod: %s on %s", pod.GetName(), pod.Spec.NodeName)
					log.Printf(" * container: %s got empty resources requests: %v, limits: %v\n",
						c.Name, c.Resources.Requests, c.Resources.Limits)
					shouldDeleteDeployment = true
					break
				}
			}
			if shouldDeleteDeployment {
				deleteDeployment(deploymentsClient, deploy.GetName())
				slackCli.notifySlack(namespace, deploy.GetName(), "empty resources constraints", "deleted")
				break
			}
		}
	}

}

func prompt() {
	log.Printf("-> Press Return key to continue.")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		break
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	log.Println()
}

func listPodsByDeployLabel(kubecli *kubernetes.Clientset, namespace string, deploy appv1.Deployment) (*apiv1.PodList, error) {
	podClient := kubecli.Core().Pods(namespace)
	set := labels.Set(deploy.Spec.Selector.MatchLabels)
	opts := metav1.ListOptions{LabelSelector: set.AsSelector().String()}
	pods, err := podClient.List(opts)
	return pods, err
}

func deleteDeployment(deploymentsClient v1.DeploymentInterface, name string) {
	// Delete Deployment
	prompt()
	log.Println("Deleting deployment...")
	deletePolicy := metav1.DeletePropagationForeground
	if err := deploymentsClient.Delete(name, &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		panic(err)
	}
	log.Println("Deleted deployment.")
}

func int32Ptr(i int32) *int32 { return &i }
