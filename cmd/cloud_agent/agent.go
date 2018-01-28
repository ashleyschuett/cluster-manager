package main

import (
	"log"
	"time"

	"github.com/containership/cloud-agent/internal/agent"
	"github.com/containership/cloud-agent/internal/k8sutil"
	"github.com/containership/cloud-agent/internal/server"
)

func main() {
	log.Println("Starting Containership agent...")

	csInformerFactory := k8sutil.CSAPI().NewCSSharedInformerFactory(time.Second * 10)

	// TODO change NewController to allow for different types (we need
	// Firewalls too eventually)
	controller := agent.NewController(k8sutil.CSAPI().Client(), csInformerFactory)

	stopCh := make(chan struct{})
	go csInformerFactory.Start(stopCh)

	// Each controller is pretty lightweight so one worker should be fine
	if err := controller.Run(1, stopCh); err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
	}

	// Run the http server
	s := server.New()
	s.Run()
}
