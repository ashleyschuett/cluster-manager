package handlers

import (
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/containership/cloud-agent/internal/envvars"
	"github.com/containership/cloud-agent/internal/k8sutil"
)

type node struct {
	corev1.NodeSystemInfo
	NodeID string `json:"nodeID"`
}

type containershipClusterMetadata struct {
	ClusterID      string `json:"cluster_id"`
	OrganizationID string `json:"organization_id"`
}
type metadata struct {
	Containership containershipClusterMetadata `json:"containership"`
	Timestamp     time.Time                    `json:"timestamp"`
	Nodes         []node
}

func getNodes() ([]node, error) {
	nodes, err := k8sutil.API().GetNodes()
	if err != nil {
		return nil, err
	}

	allNodes := make([]node, 0)
	for _, n := range nodes.Items {
		allNodes = append(allNodes, node{n.Status.NodeInfo, "nodeid"})
	}

	return allNodes, nil
}

// Get returns Containership and node metadata
func (meta *Metadata) Get(w http.ResponseWriter, r *http.Request) {
	nodes, err := getNodes()

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	m := &metadata{
		Containership: containershipClusterMetadata{
			ClusterID:      envvars.GetClusterID(),
			OrganizationID: envvars.GetOrganizationID(),
		},
		Timestamp: time.Now(),
		Nodes:     nodes,
	}

	respondWithJSON(w, http.StatusOK, m)
}
