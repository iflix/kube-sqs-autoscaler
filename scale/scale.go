package scale

import (
	"github.com/pkg/errors"

	log "github.com/Sirupsen/logrus"

	"k8s.io/kubernetes/pkg/client/restclient"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
)

type KubeClient interface {
	Deployments(namespace string) kclient.DeploymentInterface
	Scales(namespace string) kclient.ScaleInterface
}

type PodAutoScaler struct {
	Client     KubeClient
	Max        int
	Min        int
	Deployment string
	Namespace  string
}

func NewPodAutoScaler(kubernetesDeploymentName string, kubernetesNamespace string, max int, min int) *PodAutoScaler {
	config, err := restclient.InClusterConfig()
	if err != nil {
		panic("Failed to configure incluster config")
	}

	k8sClient, err := kclient.NewExtensions(config)
	if err != nil {
		panic("Failed to configure client")
	}

	return &PodAutoScaler{
		Client:     k8sClient,
		Min:        min,
		Max:        max,
		Deployment: kubernetesDeploymentName,
		Namespace:  kubernetesNamespace,
	}
}

func (p *PodAutoScaler) ScaleUp() error {
	scaleInterface := p.Client.Scales(p.Namespace)
	scale, err := scaleInterface.Get("Pod", p.Deployment)
	if err != nil {
		return errors.Wrap(err, "Failed to get scale from kube server, no scale up occurred")
	}
	currentReplicas := scale.Spec.Replicas

	if currentReplicas >= int32(p.Max) {
		return errors.New("Max pods reached")
	} else {
		log.Infof("Current replicas: %d", currentReplicas)
	}

	scale.Spec.Replicas = currentReplicas + 1

	scale, err = scaleInterface.Update("Pod", scale)
	if err != nil {
		return errors.Wrap(err, "Failed to scale up")
	}

	log.Infof("Scale up successful. Replicas: %d", scale.Spec.Replicas)
	return nil
}

func (p *PodAutoScaler) ScaleDown() error {
	scaleInterface := p.Client.Scales(p.Namespace)
	scale, err := scaleInterface.Get("Pod", p.Deployment)
	if err != nil {
		return errors.Wrap(err, "Failed to get scale from kube server, no scale up occurred")
	}
	currentReplicas := scale.Spec.Replicas

	if currentReplicas <= int32(p.Min) {
		return errors.New("Min pods reached")
	} else {
		log.Infof("Current replicas: %d", currentReplicas)
	}

	scale.Spec.Replicas = currentReplicas - 1

	scale, err = scaleInterface.Update("Pod", scale)
	if err != nil {
		return errors.Wrap(err, "Failed to scale down")
	}

	log.Infof("Scale down successful. Replicas: %d", scale.Spec.Replicas)
	return nil
}
