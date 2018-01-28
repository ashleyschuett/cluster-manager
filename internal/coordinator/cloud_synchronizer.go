package coordinator

import (
	"time"

	"github.com/containership/cloud-agent/internal/k8sutil"
	"github.com/containership/cloud-agent/internal/log"
	crdcontrollers "github.com/containership/cloud-agent/internal/resources/controller"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
)

// CloudSynchronizer synchronizes Containership Cloud resources
// into our Kubernetes CRDs.
type CloudSynchronizer struct {
	userCRDController     *crdcontrollers.UserController
	registryCRDController *crdcontrollers.RegistryController
	syncStopCh            chan struct{}
	stopped               bool
}

// NewCloudSynchronizer constructs a new CloudSynchronizer.
func NewCloudSynchronizer(csInformerFactory csinformers.SharedInformerFactory) *CloudSynchronizer {
	return &CloudSynchronizer{
		userCRDController: crdcontrollers.NewUser(
			csInformerFactory,
			k8sutil.CSAPI().Client(),
		),

		registryCRDController: crdcontrollers.NewRegistry(
			csInformerFactory,
			k8sutil.CSAPI().Client(),
		),

		syncStopCh: make(chan struct{}),
		stopped:    false,
	}
}

// Run kicks off cloud sync routines.
func (s *CloudSynchronizer) Run() {
	log.Info("Running CloudSynchronizer")
	go s.userCRDController.SyncWithCloud(s.syncStopCh)
	go s.registryCRDController.SyncWithCloud(s.syncStopCh)
}

// RequestTerminate requests that all Containership resources be deleted from
// the cluster. It kicks off a goroutine to marks resources for deletion and
// returns without blocking.
func (s *CloudSynchronizer) RequestTerminate() {
	// Stop synchronizing cloud resources
	s.stopAllSyncRoutines()

	go cleanupAllContainershipManagedResources()
}

// stopAllSyncRoutines stops all cloud synchronization but does not clean up
// any resources.
func (s *CloudSynchronizer) stopAllSyncRoutines() {
	if s.stopped {
		log.Info("CloudSynchronizer already stopped")
		return
	}

	log.Info("Stopping CloudSynchronizer")
	close(s.syncStopCh)
	s.stopped = true
}

// cleanupAllContainershipManagedResources performs a best-effort attempt to
// clean up all CS resources by deleting all CS CRDs and then deleting the core
// Containership namespace after a delay, which should result in the agent and
// coordinator being killed.
func cleanupAllContainershipManagedResources() {
	tryDeleteAllContainershipCRDs()

	// TODO it would be great if we could avoid this arbitrary heuristic here.
	// Sleep for a little to give k8s enough time to attempt to delete all
	// resources. This should provide enough time for the agents to see the
	// CRD deletions (which should happen first) and clean up any on-host changes
	// such as authorized_keys.
	time.Sleep(time.Minute)

	tryDeleteAllContainershipNamespaces()
}

// tryDeleteAllContainershipCRDs tries to delete all CRDs managed by us. This will cause cascading delete to clean up all Containership-managed resources.
func tryDeleteAllContainershipCRDs() {
	crdList, err := k8sutil.ExtensionsAPI().GetContainershipCRDs()
	if err != nil {
		log.Error("Could not list CRDs for cleanup:", err.Error())
		return
	}

	for _, crd := range crdList.Items {
		log.Info("Deleting CRD", crd.Name)
		err := k8sutil.ExtensionsAPI().DeleteCRD(crd.Name)
		if err != nil {
			log.Errorf("Could not delete CRD %s: %s", crd.Name, err.Error())
		}
	}
}

// tryDeleteAllContainershipNamespaces tries to delete all
// namespaces managed by us. This will cause all resources
// belonging to the namespace, including the pod that this is
// running in, to be deleted.
func tryDeleteAllContainershipNamespaces() {
	nsList, err := k8sutil.API().GetContainershipNamespaces()
	if err != nil {
		log.Error("Could not list namespaces for cleanup:", err.Error())
		return
	}

	for _, ns := range nsList.Items {
		log.Info("Deleting Namespace", ns.Name)
		err := k8sutil.API().DeleteNamespace(ns.Name)
		if err != nil {
			log.Errorf("Could not delete Namespace %s: %s", ns.Name, err.Error())
		}
	}
}
