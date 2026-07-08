package config

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/GoogleCloudPlatform/ai-on-gke/static-np-provisioner/internal/cloud"
	corev1 "k8s.io/api/core/v1"
)

var topologyRegex = regexp.MustCompile(`^(\d+)x(\d+)x(\d+)$`)

func ValidateConfigMap(cm *corev1.ConfigMap) error {
	reservations, defaultConfig, err := ParseConfigMap(cm)
	if err != nil {
		return fmt.Errorf("parsing configuration: %w", err)
	}

	for _, res := range reservations {
		if res.Name == "" {
			return fmt.Errorf("reservation name cannot be empty")
		}

		if len(res.GscSubblocks) == 0 {
			return fmt.Errorf("reservation %s: gscSubblocks list cannot be empty", res.Name)
		}

		for _, subblock := range res.GscSubblocks {
			if subblock.Block == "" {
				return fmt.Errorf("reservation %s: block name cannot be empty", res.Name)
			}

			if subblock.Subblocks == "" {
				return fmt.Errorf("reservation %s (block %s): subblocks cannot be empty", res.Name, subblock.Block)
			}

			// Validate subblock range format
			start, end, err := cloud.ParseSubBlocks(subblock.Subblocks)
			if err != nil {
				return fmt.Errorf("reservation %s (block %s): invalid subblock range %q: %w", res.Name, subblock.Block, subblock.Subblocks, err)
			}
			if start < 0 || end < 0 {
				return fmt.Errorf("reservation %s (block %s): subblock indexes cannot be negative", res.Name, subblock.Block)
			}

			// Determine fully merged config for the subblock
			var subblockConfig *cloud.StaticNodePoolConfig
			if subblock.NodepoolConfig != nil {
				subblockConfig = MergeConfig(defaultConfig, subblock.NodepoolConfig)
			} else {
				subblockConfig = defaultConfig
			}

			if subblockConfig == nil {
				return fmt.Errorf("reservation %s (block %s): no nodepool config specified (neither subblock-level nor default)", res.Name, subblock.Block)
			}

			if err := validateNodepoolConfig(subblockConfig); err != nil {
				return fmt.Errorf("reservation %s (block %s): %w", res.Name, subblock.Block, err)
			}
		}
	}

	return nil
}

var supportedTopologies = map[string]bool{
	"2x2x1": true,
	"2x2x2": true,
	"2x2x4": true,
	"2x4x4": true,
	"4x4x4": true,
}

func validateNodepoolConfig(config *cloud.StaticNodePoolConfig) error {
	if config.MachineType == "" {
		return fmt.Errorf("machineType cannot be empty")
	}

	if config.MachineType != "tpu7x-standard-4t" {
		return fmt.Errorf("unsupported machineType %q: static provisioner strictly supports 'tpu7x-standard-4t'", config.MachineType)
	}

	if config.Topology == "" {
		return fmt.Errorf("topology cannot be empty")
	}

	if config.PlacementPolicy == "" {
		return fmt.Errorf("placementPolicy cannot be empty")
	}

	if config.NodeCount <= 0 {
		return fmt.Errorf("nodeCount must be greater than 0 (got %d)", config.NodeCount)
	}

	// 1. Validate 3D topology format
	matches := topologyRegex.FindStringSubmatch(config.Topology)
	if matches == nil {
		return fmt.Errorf("invalid topology format %q: must be a 3D shape like '2x2x1' or '4x4x4'", config.Topology)
	}

	// 2. Check if the topology is supported by this provisioner (strictly up to 4x4x4 topology)
	if !supportedTopologies[config.Topology] {
		return fmt.Errorf("topology %q is not supported by this provisioner. Supported topologies are: 2x2x1, 2x2x2, 2x2x4, 2x4x4, 4x4x4", config.Topology)
	}

	// 3. Calculate total chips from 3D topology dimensions
	totalChips := 1
	for _, dimStr := range matches[1:] {
		dim, err := strconv.Atoi(dimStr)
		if err != nil {
			return fmt.Errorf("parsing topology dimension %q: %w", dimStr, err)
		}
		totalChips *= dim
	}

	// GKE TPU v7x hardware architecture strictly packs exactly 4 chips per VM host
	const chipsPerHost = 4

	// 4. Validate nodeCount matches required chips / host
	if totalChips%chipsPerHost != 0 {
		return fmt.Errorf("topology %q (total chips: %d) is incompatible with chips per host (%d) for machine type %q",
			config.Topology, totalChips, chipsPerHost, config.MachineType)
	}

	expectedNodeCount := totalChips / chipsPerHost
	if config.NodeCount != expectedNodeCount {
		return fmt.Errorf("topology %q (total chips: %d) with machine type %q requires exactly %d nodes, but nodeCount is set to %d",
			config.Topology, totalChips, config.MachineType, expectedNodeCount, config.NodeCount)
	}

	return nil
}
