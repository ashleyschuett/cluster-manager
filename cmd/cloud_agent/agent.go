package main

import (
	"runtime"

	"github.com/containership/cluster-manager/pkg/agent"
	"github.com/containership/cluster-manager/pkg/buildinfo"
	"github.com/containership/cluster-manager/pkg/env"
	"github.com/containership/cluster-manager/pkg/log"
)

func main() {
	log.Info("Starting Containership Cloud Agent...")
	log.Infof("Version: %s", buildinfo.String())
	log.Infof("Go Version: %s", runtime.Version())

	env.Dump()

	agent.Initialize()
	go agent.Run()

	// Note that we'll never actually exit because some goroutines out of our
	// control (e.g. the glog flush daemon) will continue to run).
	runtime.Goexit()
}
