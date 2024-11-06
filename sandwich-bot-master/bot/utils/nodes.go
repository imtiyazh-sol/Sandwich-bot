package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type Node struct {
	URL string `json:"url"`
}

func (n Node) IsAvailable() bool {
	return true
}

func ReadNodes(filepath string, network string, testnet bool) ([]Node, error) {
	file, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatalf("Unable to read nodes.json: %v", err)
	}

	var nodes map[string][]Node
	err = json.Unmarshal(file, &nodes)
	if err != nil {
		log.Fatalf("Unable to unmarshal nodes.json: %v", err)
	}

	networkNodes, exists := nodes[network]
	if !exists || len(networkNodes) == 0 {
		return nil, fmt.Errorf("no nodes found for network: %s", network)
	}

	return networkNodes, nil
}
