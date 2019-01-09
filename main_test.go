package main

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/watch"

	"github.com/Wattpad/kube-sqs-autoscaler/scale"
	mainsqs "github.com/Wattpad/kube-sqs-autoscaler/sqs"
)

func TestRunReachMinReplicas(t *testing.T) {
	// override default vars for testing
	pollInterval = 1 * time.Second
	scaleDownCoolPeriod = 1 * time.Second
	scaleUpCoolPeriod = 1 * time.Second
	scaleUpMessages = 100
	scaleDownMessages = 10
	maxPods = 5
	minPods = 1
	awsRegion = "us-east-1"

	sqsQueueUrl = "example.com"
	kubernetesDeploymentName = "test"
	kubernetesNamespace = "test"

	p := NewMockPodAutoScaler(kubernetesDeploymentName, kubernetesNamespace, maxPods, minPods)
	s := NewMockSqsClient()

	go Run(p, s)

	Attributes := map[string]*string{"ApproximateNumberOfMessages": aws.String("10")}
	input := &sqs.SetQueueAttributesInput{
		Attributes: Attributes,
	}
	s.Client.SetQueueAttributes(input)

	time.Sleep(10 * time.Second)
	scaleSpec, _ := p.Client.Scales("test").Get("Deployment", "test")
	assert.Equal(t, int32(minPods), scaleSpec.Spec.Replicas, "Number of replicas should be the min")
}

func TestRunReachMaxReplicas(t *testing.T) {
	// override default vars for testing
	pollInterval = 1 * time.Second
	scaleDownCoolPeriod = 1 * time.Second
	scaleUpCoolPeriod = 1 * time.Second
	scaleUpMessages = 100
	scaleDownMessages = 10
	maxPods = 5
	minPods = 1
	awsRegion = "us-east-1"

	sqsQueueUrl = "example.com"
	kubernetesDeploymentName = "test"
	kubernetesNamespace = "test"

	p := NewMockPodAutoScaler(kubernetesDeploymentName, kubernetesNamespace, maxPods, minPods)
	s := NewMockSqsClient()

	go Run(p, s)

	Attributes := map[string]*string{"ApproximateNumberOfMessages": aws.String("100")}

	input := &sqs.SetQueueAttributesInput{
		Attributes: Attributes,
	}
	s.Client.SetQueueAttributes(input)

	time.Sleep(10 * time.Second)
	scaleSpec, _ := p.Client.Scales("test").Get("Deployment", "test")
	assert.Equal(t, int32(maxPods), scaleSpec.Spec.Replicas, "Number of replicas should be the max")
}

func TestRunScaleUpCoolDown(t *testing.T) {
	pollInterval = 5 * time.Second
	scaleDownCoolPeriod = 10 * time.Second
	scaleUpCoolPeriod = 10 * time.Second
	scaleUpMessages = 100
	scaleDownMessages = 10
	maxPods = 5
	minPods = 1
	awsRegion = "us-east-1"

	sqsQueueUrl = "example.com"
	kubernetesDeploymentName = "test"

	p := NewMockPodAutoScaler(kubernetesDeploymentName, kubernetesNamespace, maxPods, minPods)
	s := NewMockSqsClient()

	go Run(p, s)

	Attributes := map[string]*string{"ApproximateNumberOfMessages": aws.String("100")}

	input := &sqs.SetQueueAttributesInput{
		Attributes: Attributes,
	}
	s.Client.SetQueueAttributes(input)

	time.Sleep(15 * time.Second)
	scaleSpec, _ := p.Client.Scales("test").Get("Deployment", "test")
	assert.Equal(t, int32(4), scaleSpec.Spec.Replicas, "Number of replicas should be 4 if cool down for scaling up was obeyed")
}

func TestRunScaleDownCoolDown(t *testing.T) {
	pollInterval = 5 * time.Second
	scaleDownCoolPeriod = 10 * time.Second
	scaleUpCoolPeriod = 10 * time.Second
	scaleUpMessages = 100
	scaleDownMessages = 10
	maxPods = 5
	minPods = 1
	awsRegion = "us-east-1"

	sqsQueueUrl = "example.com"
	kubernetesDeploymentName = "test"
	kubernetesNamespace = "test"

	p := NewMockPodAutoScaler(kubernetesDeploymentName, kubernetesNamespace, maxPods, minPods)
	s := NewMockSqsClient()

	go Run(p, s)

	Attributes := map[string]*string{"ApproximateNumberOfMessages": aws.String("10")}

	input := &sqs.SetQueueAttributesInput{
		Attributes: Attributes,
	}
	s.Client.SetQueueAttributes(input)

	time.Sleep(15 * time.Second)
	scaleSpec, _ := p.Client.Scales("test").Get("Deployment", "test")
	assert.Equal(t, int32(2), scaleSpec.Spec.Replicas, "Number of replicas should be 2 if cool down for scaling down was obeyed")
}

type MockDeployment struct {
	client *MockKubeClient
}

type MockKubeClient struct {
	// stores the state of Deployment as if the api server did
	Deployment *extensions.Deployment
	Scale      *extensions.Scale
}

type MockScale struct {
	client *MockKubeClient
}

func (m *MockDeployment) Get(name string) (*extensions.Deployment, error) {
	return m.client.Deployment, nil
}

func (m *MockDeployment) Update(deployment *extensions.Deployment) (*extensions.Deployment, error) {
	m.client.Deployment.Spec.Replicas = deployment.Spec.Replicas
	return m.client.Deployment, nil
}

func (m *MockDeployment) List(opts api.ListOptions) (*extensions.DeploymentList, error) {
	return nil, nil
}

func (m *MockDeployment) Delete(name string, options *api.DeleteOptions) error {
	return nil
}

func (m *MockDeployment) Create(*extensions.Deployment) (*extensions.Deployment, error) {
	return nil, nil
}

func (m *MockDeployment) UpdateStatus(*extensions.Deployment) (*extensions.Deployment, error) {
	return nil, nil
}

func (m *MockDeployment) Watch(opts api.ListOptions) (watch.Interface, error) {
	return nil, nil
}

func (m *MockDeployment) Rollback(*extensions.DeploymentRollback) error {
	return nil
}

func (m *MockKubeClient) Deployments(namespace string) kclient.DeploymentInterface {
	return &MockDeployment{
		client: m,
	}
}

func (m *MockScale) Get(kind string, deployment string) (*extensions.Scale, error) {
	return m.client.Scale, nil
}

func (m *MockScale) Update(kind string, scale *extensions.Scale) (*extensions.Scale, error) {
	m.client.Scale.Spec.Replicas = scale.Spec.Replicas
	return m.client.Scale, nil
}

func (m *MockKubeClient) Scales(namespace string) kclient.ScaleInterface {
	return &MockScale{
		client: m,
	}
}

func NewMockKubeClient() *MockKubeClient {
	return &MockKubeClient{
		Deployment: &extensions.Deployment{
			Spec: extensions.DeploymentSpec{
				Replicas: 3,
			},
		},
		Scale: &extensions.Scale{
			Spec: extensions.ScaleSpec{
				Replicas: 3,
			},
		},
	}
}

func NewMockPodAutoScaler(kubernetesDeploymentName string, kubernetesNamespace string, max int, min int) *scale.PodAutoScaler {
	mockClient := NewMockKubeClient()

	return &scale.PodAutoScaler{
		Client:     mockClient,
		Min:        min,
		Max:        max,
		Deployment: kubernetesDeploymentName,
		Namespace:  kubernetesNamespace,
	}
}

type MockSQS struct {
	QueueAttributes *sqs.GetQueueAttributesOutput
}

func (m *MockSQS) GetQueueAttributes(*sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error) {
	return m.QueueAttributes, nil
}

func (m *MockSQS) SetQueueAttributes(input *sqs.SetQueueAttributesInput) (*sqs.SetQueueAttributesOutput, error) {
	m.QueueAttributes = &sqs.GetQueueAttributesOutput{
		Attributes: input.Attributes,
	}
	return &sqs.SetQueueAttributesOutput{}, nil
}

func NewMockSqsClient() *mainsqs.SqsClient {
	Attributes := map[string]*string{"ApproximateNumberOfMessages": aws.String("50")}

	return &mainsqs.SqsClient{
		Client: &MockSQS{
			QueueAttributes: &sqs.GetQueueAttributesOutput{
				Attributes: Attributes,
			},
		},
		QueueUrl: "example.com",
	}
}
