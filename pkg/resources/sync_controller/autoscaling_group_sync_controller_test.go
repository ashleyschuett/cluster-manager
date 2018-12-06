package synccontroller

import (
	"testing"
	"time"

	"k8s.io/client-go/kubernetes/fake"

	fakecerebral "github.com/containership/cerebral/pkg/client/clientset/versioned/fake"
	cerebralinformers "github.com/containership/cerebral/pkg/client/informers/externalversions"
	"github.com/stretchr/testify/assert"
)

func initalizeFakeAutoscalingGroupController() *AutoscalingGroupSyncController {
	client := &fake.Clientset{}
	cerebralclient := fakecerebral.NewSimpleClientset()

	cerebralInformerFactory := cerebralinformers.NewSharedInformerFactory(cerebralclient, 30*time.Second)
	return NewAutoscalingGroupController(client, cerebralclient, cerebralInformerFactory)
}

func TestNewAutoscalingGroupController(t *testing.T) {
	agc := initalizeFakeAutoscalingGroupController()

	assert.Equal(t, autoscalingGroupSyncControllerName, agc.syncController.name)
}
