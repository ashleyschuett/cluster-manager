package main

import (
	"fmt"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/containership/cloud-agent/internal/constants"
	"github.com/containership/cloud-agent/internal/k8sutil"
	"github.com/containership/cloud-agent/internal/log"

	provisioncsv3 "github.com/containership/cloud-agent/pkg/apis/provision.containership.io/v3"
)

func main() {
	// TODO options instead of hardcoding everything
	if err := run(); err != nil {
		log.Errorf("Test failed: %s", err)
		os.Exit(1)
	}

	log.Infof("Test passed!")
	os.Exit(0)
}

const (
	pollIntervalSeconds      = 5
	nodeTimeoutSeconds       = 90
	nodeTimeoutBufferSeconds = 20
)

// TODO put this somewhere useful
func createClusterUpgrade(targetVersion string, id string) (*provisioncsv3.ClusterUpgrade, error) {
	log.Infof("Creating ClusterUpgrade %q with target version %s", id, targetVersion)

	labels := constants.BuildContainershipLabelMap(nil)

	cup, err := k8sutil.CSAPI().Client().ContainershipProvisionV3().ClusterUpgrades(constants.ContainershipNamespace).Create(&provisioncsv3.ClusterUpgrade{
		ObjectMeta: metav1.ObjectMeta{
			Name:   id,
			Labels: labels,
		},
		Spec: provisioncsv3.ClusterUpgradeSpec{
			ID:                 id,
			Type:               provisioncsv3.UpgradeTypeKubernetes,
			AddedAt:            "TODO",
			Description:        id,
			TargetVersion:      targetVersion,
			LabelSelector:      nil, // all nodes
			NodeTimeoutSeconds: nodeTimeoutSeconds,
		}})

	return cup, err
}

// TODO put this somewhere useful
func deleteClusterUpgrade(upgradeName string) error {
	log.Infof("Deleting ClusterUpgrade %q", upgradeName)
	return k8sutil.CSAPI().Client().ContainershipProvisionV3().ClusterUpgrades(constants.ContainershipNamespace).Delete(upgradeName, &metav1.DeleteOptions{})
}

// TODO polling is really stupid. We should use a watch / informer, but that
// wasn't working for some reason
func pollUpgrade(upgradeName string) provisioncsv3.UpgradeStatus {
	for {
		cup, err := k8sutil.CSAPI().Client().ContainershipProvisionV3().
			ClusterUpgrades(constants.ContainershipNamespace).Get(upgradeName, metav1.GetOptions{})
		if err != nil {
			// TODO there's some weird ephemeral permissions error that happens
			// and breaks things, just retry for now
			continue
		}

		// If the overall status is done then we're done - return that status
		log.Debugf("Cluster upgrade %q has cluster status %q", upgradeName, cup.Spec.Status.ClusterStatus)
		switch cup.Spec.Status.ClusterStatus {
		case provisioncsv3.UpgradeSuccess, provisioncsv3.UpgradeFailed:
			log.Infof("Cluster upgrade %q finished with cluster status %q", upgradeName, cup.Spec.Status.ClusterStatus)
			return cup.Spec.Status.ClusterStatus
		}

		// If an individual node failed then return that (fail fast)
		// Also check if any node is in progress for additional checking later
		nodeInProgress := ""
		for nodeName, nodeStatus := range cup.Spec.Status.NodeStatuses {
			if nodeStatus == provisioncsv3.UpgradeFailed {
				log.Errorf("Cluster upgrade %q failed for node %q", upgradeName, nodeName)
				return nodeStatus
			}

			if nodeStatus == provisioncsv3.UpgradeInProgress {
				log.Debugf("Cluster upgrade %q has InProgress node %q", upgradeName, nodeName)
				nodeInProgress = nodeName
			}
		}

		// Check for timeout - if a node times out, then its status should be
		// updated to Failed after nodeTimeoutSeconds, but we're testing that
		// functionality :)
		// Add some additional buffer to the real timeout to give it a chance
		// to fail properly
		// Note that this assumes that the start time is set properly in the status
		if nodeInProgress != "" {
			startTime, _ := time.Parse(time.UnixDate, cup.Spec.Status.CurrentStartTime)
			timeoutDuration := (nodeTimeoutSeconds + nodeTimeoutBufferSeconds) * time.Second
			if !startTime.IsZero() && time.Since(startTime) > timeoutDuration {
				log.Errorf("Upgrade %q failed to succeed or time out for node %q", upgradeName, nodeInProgress)
				return provisioncsv3.UpgradeFailed
			}
		}

		// Nothing interesting, keep chugging
		time.Sleep(pollIntervalSeconds * time.Second)
	}
}

// TODO arguments
func run() error {
	for i := 0; ; i++ {
		// TODO list of versions to cycle through as arg, don't just hardcode
		// and flip between two
		targetVersion := "v1.10.2"
		if i%2 != 0 {
			targetVersion = "v1.10.1"
		}

		seq := 0
		id := fmt.Sprintf("%s-%d", targetVersion, seq)
		_, err := createClusterUpgrade(targetVersion, id)
		if err != nil {
			return err
		}

		log.Infof("Polling upgrade %q until done", id)
		if result := pollUpgrade(id); result == provisioncsv3.UpgradeFailed {
			err = fmt.Errorf("Upgrade %q failed", id)
		}

		// Clean up
		deleteClusterUpgrade(id)

		if err != nil {
			return err
		}
	}
}
