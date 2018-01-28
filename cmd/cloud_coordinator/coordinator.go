package main

import (
	"runtime"
	"time"

	"github.com/containership/cloud-agent/internal/coordinator"
	"github.com/containership/cloud-agent/internal/k8sutil"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/server"

	crdcontrollers "github.com/containership/cloud-agent/internal/resources/controller"
)

func main() {
	log.Info("Starting Containership coordinator...")

	// Kick off CS-->k8s synchronization
	stopCh := make(chan struct{})

	// Create Informer factories. All Informers should be created from these
	// factories in order to share the same underlying caches.
	kubeInformerFactory := k8sutil.API().NewKubeSharedInformerFactory(time.Second * 10)
	csInformerFactory := k8sutil.CSAPI().NewCSSharedInformerFactory(time.Second * 10)

	controller := coordinator.NewController(
		k8sutil.API().Client(), k8sutil.CSAPI().Client(), kubeInformerFactory, csInformerFactory)

	// CRD controllers need to be created before any jobs start so that all
	// needed index functions can be added to the informers
	userCRDcontroller := crdcontrollers.NewUser(
		csInformerFactory,
		k8sutil.CSAPI().Client(),
	)

	registryCRDcontroller := crdcontrollers.NewRegistry(
		csInformerFactory,
		k8sutil.CSAPI().Client(),
	)

	// Kick off the informer factories
	kubeInformerFactory.Start(stopCh)
	csInformerFactory.Start(stopCh)

	go userCRDcontroller.SyncWithCloud(stopCh)
	go registryCRDcontroller.SyncWithCloud(stopCh)

	// Run the http server
	s := server.New()
	go s.Run()

	if err := controller.Run(2, stopCh); err != nil {
		log.Fatal("Error running controller:", err.Error())
	}

	runtime.Goexit()
}
