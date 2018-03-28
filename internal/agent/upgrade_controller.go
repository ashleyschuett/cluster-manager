package agent

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/containership/cloud-agent/internal/constants"
	"github.com/containership/cloud-agent/internal/env"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/request"
	"github.com/containership/cloud-agent/internal/resources/upgradescript"
	"github.com/containership/cloud-agent/internal/tools"

	provisioncsv3 "github.com/containership/cloud-agent/pkg/apis/provision.containership.io/v3"
	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
	pcslisters "github.com/containership/cloud-agent/pkg/client/listers/provision.containership.io/v3"
)

const (
	upgradeControllerName = "UpgradeAgentController"

	maxRetriesUpgradeController = 5
)

const (
	// TODO finalize this - current version is just for rough testing
	nodeUpgradeScriptEndpointTemplate = "/organizations/{{.OrganizationID}}/clusters/{{.ClusterID}}/nodes/{{.NodeName}}-upgrade.sh"
)

// UpgradeController is the agent controller which watches for ClusterUpgrade updates
// and writes update script to host when it is that specific agents turn to update
type UpgradeController struct {
	clientset     csclientset.Interface
	kubeclientset kubernetes.Interface

	upgradeLister  pcslisters.ClusterUpgradeLister
	upgradesSynced cache.InformerSynced
	nodeLister     corelistersv1.NodeLister
	nodesSynced    cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
}

// NewUpgradeController creates a new agent UpgradeController
func NewUpgradeController(
	clientset csclientset.Interface,
	csInformerFactory csinformers.SharedInformerFactory,
	kubeclientset kubernetes.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory) *UpgradeController {

	uc := &UpgradeController{
		clientset:     clientset,
		kubeclientset: kubeclientset,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), upgradeControllerName),
	}

	// Create an informer from the factory so that we share the underlying
	// cache with other controllers
	upgradeInformer := csInformerFactory.ContainershipProvision().V3().ClusterUpgrades()
	nodeInformer := kubeInformerFactory.Core().V1().Nodes()

	// All event handlers simply add to a workqueue to be processed by a worker
	upgradeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			oldUpgrade := old.(*provisioncsv3.ClusterUpgrade)
			newUpgrade := new.(*provisioncsv3.ClusterUpgrade)
			if oldUpgrade.ResourceVersion == newUpgrade.ResourceVersion ||
				newUpgrade.Spec.CurrentNode != env.NodeName() {
				// syncInterval update or not the current node so nothing to do
				return
			}
			uc.enqueueUpgrade(new)
		},
	})

	// We need to trigger synchronization on node annotation updates
	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			oldNode := old.(*corev1.Node)
			newNode := new.(*corev1.Node)
			if oldNode.ResourceVersion == newNode.ResourceVersion ||
				newNode.Name != env.NodeName() {
				// syncInterval update or not the current node so nothing to do
				return
			}
			uc.enqueueNode(new)
		},
	})

	uc.upgradeLister = upgradeInformer.Lister()
	uc.upgradesSynced = upgradeInformer.Informer().HasSynced
	uc.nodeLister = nodeInformer.Lister()
	uc.nodesSynced = nodeInformer.Informer().HasSynced

	return uc
}

// Run kicks off the Controller with the given number of workers to process the
// workqueue
func (uc *UpgradeController) Run(numWorkers int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer uc.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info("Starting Upgrade controller")

	log.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, uc.upgradesSynced, uc.nodesSynced); !ok {
		return fmt.Errorf("Failed to wait for caches to sync")
	}

	log.Info("Starting upgrade workers")
	// Launch numWorkers workers to process Upgrade resource
	for i := 0; i < numWorkers; i++ {
		go wait.Until(uc.runWorker, time.Second, stopCh)
	}

	log.Info("Started upgrade workers")
	<-stopCh
	log.Info("Shutting down upgrade controller")

	return nil
}

// runWorker continually requests that the next queue item be processed
func (uc *UpgradeController) runWorker() {
	for uc.processNextWorkItem() {
	}
}

// processNextWorkItem continually pops items off of the workqueue and handles
// them
func (uc *UpgradeController) processNextWorkItem() bool {
	obj, shutdown := uc.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer uc.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			uc.workqueue.Forget(obj)
			log.Errorf("expected string in workqueue but got %#v", obj)
			return nil
		}

		// A common syncHandler is called for keys of any type (node or upgrade)
		err := uc.syncHandler(key)
		return uc.handleErr(err, key)
	}(obj)

	if err != nil {
		log.Error(err)
		return true
	}

	return true
}

// handleErr looks to see if the resource sync event returned with an error,
// if it did the resource gets requeued up to as many times as is set for
// the max retries. If retry count is hit, or the resource is synced successfully
// the resource is moved off the queue
func (uc *UpgradeController) handleErr(err error, key interface{}) error {
	if err == nil {
		uc.workqueue.Forget(key)
		return nil
	}

	if uc.workqueue.NumRequeues(key) < maxRetriesUpgradeController {
		uc.workqueue.AddRateLimited(key)
		return fmt.Errorf("error syncing '%v': %s. Has been resynced %v times", key, err.Error(), uc.workqueue.NumRequeues(key))
	}

	uc.workqueue.Forget(key)
	log.Infof("Dropping %v out of the queue: %v", key, err)
	return err
}

// enqueueUpgrade enqueues an upgrade
func (uc *UpgradeController) enqueueUpgrade(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Error(err)
		return
	}

	uc.workqueue.AddRateLimited(key)
}

// enqueueNode enqueues a node
func (uc *UpgradeController) enqueueNode(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Error(err)
		return
	}

	uc.workqueue.AddRateLimited(key)
}

// syncHandler looks at the current state of the system and decides how to act.
// For upgrade that means writing the upgrade script to the directory that is being
// watched by the systemd upgrade process.
func (uc *UpgradeController) syncHandler(key string) error {
	log.Infof("%s: processing key=%q", upgradeControllerName, key)

	upgrade, err := uc.getCurrentUpgrade()
	if err != nil {
		return err
	}
	if upgrade == nil {
		// No active upgrades, so nothing to do
	}

	node, _ := uc.nodeLister.Get(env.NodeName())

	upgradeAnnotation, err := tools.GetNodeUpgradeAnnotation(node)
	if err != nil {
		return err
	}

	switch {
	case upgradeAnnotation == nil:
		// Upgrade hasn't started or we're in an unknown state
		if tools.NodeIsTargetKubernetesVersion(upgrade, node) {
			// Nothing to do since we're at the desired version
			return nil
		}

		// Kick off upgrade and mark status as InProgress
		return uc.startUpgrade(node, upgrade)

	case upgradeAnnotation.Status == provisioncsv3.UpgradeInProgress:
		if tools.NodeIsTargetKubernetesVersion(upgrade, node) {
			// We must be done upgrading - presumably it succeeded because
			// we're at the version we expect and we didn't time out.
			// Finish the upgrade with a Success status.
			return uc.finishUpgradeWithStatus(node, provisioncsv3.UpgradeSuccess)
		}

		// TODO add timeout logic so upgrade fails after set period of time

	case upgradeAnnotation.Status == provisioncsv3.UpgradeSuccess:
		fallthrough
	case upgradeAnnotation.Status == provisioncsv3.UpgradeFailed:
		fallthrough
	default:
		// Nothing to do
		return nil
	}

	return nil
}

// getCurrentUpgrade returns the current in-progress upgrade or nil if no upgrade
// is in-progress.
// TODO need to enforce only one InProgress upgrade at a time
// TODO shared function between agent and coordinator controllers
func (uc *UpgradeController) getCurrentUpgrade() (*provisioncsv3.ClusterUpgrade, error) {
	upgrades, err := uc.upgradeLister.ClusterUpgrades(constants.ContainershipNamespace).
		List(constants.GetContainershipManagedSelector())
	if err != nil {
		return nil, err
	}

	for _, upgrade := range upgrades {
		if upgrade.Spec.Status == provisioncsv3.UpgradeInProgress {
			return upgrade, nil
		}
	}

	return nil, nil
}

// startUpgrade kicks off the upgrade process by downloading and writing the
// upgrade script as well as updating the current node's upgrade status.
func (uc *UpgradeController) startUpgrade(node *corev1.Node, upgrade *provisioncsv3.ClusterUpgrade) error {
	log.Info("Beginning upgrade process")

	// Step 1: Fetch the upgrade script from Cloud
	log.Info("Downloading upgrade script")
	script, err := uc.downloadUpgradeScript()
	if err != nil {
		log.Error("Download upgrade script failed:", err)
		return err
	}

	// Step 2: Mark node as in progress
	log.Info("Setting node upgrade status to InProgress")
	err = uc.writeNodeUpgradeAnnotation(node,
		provisioncsv3.NodeUpgradeAnnotation{
			upgrade.Spec.TargetKubernetesVersion,
			provisioncsv3.UpgradeInProgress,
			time.Now().UTC(),
		})
	if err != nil {
		log.Error("Node annotation write failed:", err)
		return err
	}

	// Step 3: Execute the upgrade script
	log.Info("Writing upgrade script")
	targetVersion := upgrade.Spec.TargetKubernetesVersion
	upgradeID := upgrade.Spec.ID
	return upgradescript.Write(script, targetVersion, upgradeID)
}

// finishUpgradeWithStatus performs any necessary cleanup and posts the updated
// status for this node.
func (uc *UpgradeController) finishUpgradeWithStatus(node *corev1.Node,
	status provisioncsv3.UpgradeStatus) error {
	log.Info("Setting node upgrade status to %s", string(status))

	// Remove the `current` file first so regardless of any failures after this point
	// we'll be able to retry if needed by writing a new `current`
	if err := upgradescript.RemoveCurrent(); err != nil {
		// There's no good option for handling this, so just continue instead of
		// failing the upgrade.
		log.Error("Could not remove `current` upgrade file:", err)
	}

	upgradeAnnotation, err := tools.GetNodeUpgradeAnnotation(node)
	if err != nil {
		return err
	}

	return uc.writeNodeUpgradeAnnotation(node,
		provisioncsv3.NodeUpgradeAnnotation{
			upgradeAnnotation.ClusterVersion,
			status,
			upgradeAnnotation.StartTime,
		})
}

func (uc *UpgradeController) downloadUpgradeScript() ([]byte, error) {
	req, err := request.New(request.CloudServiceProvision,
		nodeUpgradeScriptEndpointTemplate,
		"GET",
		nil)
	if err != nil {
		return nil, err
	}

	resp, err := req.MakeRequest()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// writeNodeUpgradeAnnotation writes the node upgrade annotation
// struct to the node
func (uc *UpgradeController) writeNodeUpgradeAnnotation(node *corev1.Node, upgradeAnnotation provisioncsv3.NodeUpgradeAnnotation) error {
	node = node.DeepCopy()

	annotBytes, err := json.Marshal(upgradeAnnotation)
	if err != nil {
		return err
	}

	annotations := node.ObjectMeta.Annotations
	annotations[constants.NodeUpgradeAnnotationKey] = string(annotBytes)

	_, err = uc.kubeclientset.Core().Nodes().Update(node)

	return err
}
