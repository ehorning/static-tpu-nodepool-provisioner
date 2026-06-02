package config

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultNodepoolConfig4x4x4 = `
nodepoolPrefix: "example"
machineType: "tpu7x-standard-4t"
topology: "4x4x4"
placementPolicy: "tpu-provisioner-4x4x4"
nodeCount: 16
`
)

func TestValidateConfigMap(t *testing.T) {
	tests := []struct {
		name            string
		cmData          map[string]string
		wantErr         bool
		wantErrContains string
	}{
		{
			name: "Valid config with default fallbacks",
			cmData: map[string]string{
				"reservations": `
- name: "my-gsc-reservation"
  gscSubblocks:
    - block: "block-1"
      subblocks: "0001-0004"
`,
				"defaultNodepoolConfig": defaultNodepoolConfig4x4x4,
			},
			wantErr: false,
		},
		{
			name: "Valid config with subblock overrides",
			cmData: map[string]string{
				"reservations": `
- name: "my-gsc-reservation"
  gscSubblocks:
    - block: "block-1"
      subblocks: "0001-0002"
      nodepoolConfig:
        machineType: "tpu7x-standard-4t"
        topology: "2x2x2"
        placementPolicy: "tpu-provisioner-2x2x2"
        nodeCount: 2
`,
				"defaultNodepoolConfig": defaultNodepoolConfig4x4x4,
			},
			wantErr: false,
		},
		{
			name: "Invalid YAML structure",
			cmData: map[string]string{
				"reservations": `
- name: "my-gsc-reservation"
  gscSubblocks:
    - block: "block-1"
      subblocks: [unclosed brackets
`,
			},
			wantErr:         true,
			wantErrContains: "parsing configuration",
		},
		{
			name: "Missing reservations key",
			cmData: map[string]string{
				"defaultNodepoolConfig": defaultNodepoolConfig4x4x4,
			},
			wantErr:         true,
			wantErrContains: "no 'reservations' key in configmap",
		},
		{
			name: "Empty reservation name",
			cmData: map[string]string{
				"reservations": `
- name: ""
  gscSubblocks:
    - block: "block-1"
      subblocks: "0001"
`,
				"defaultNodepoolConfig": defaultNodepoolConfig4x4x4,
			},
			wantErr:         true,
			wantErrContains: "reservation name cannot be empty",
		},
		{
			name: "Invalid subblock range format",
			cmData: map[string]string{
				"reservations": `
- name: "my-gsc-res"
  gscSubblocks:
    - block: "block-1"
      subblocks: "0005-0001"
`,
				"defaultNodepoolConfig": defaultNodepoolConfig4x4x4,
			},
			wantErr:         true,
			wantErrContains: "invalid subblock range",
		},
		{
			name: "Invalid topology format",
			cmData: map[string]string{
				"reservations": `
- name: "my-gsc-res"
  gscSubblocks:
    - block: "block-1"
      subblocks: "0001"
`,
				"defaultNodepoolConfig": `
machineType: "tpu7x-standard-4t"
topology: "2x2x2x2"
placementPolicy: "tpu-provisioner-2x2x2x2"
nodeCount: 1
`,
			},
			wantErr:         true,
			wantErrContains: "invalid topology format",
		},
		{
			name: "Rejected config with 2D topology",
			cmData: map[string]string{
				"reservations": `
- name: "my-gsc-res"
  gscSubblocks:
    - block: "block-1"
      subblocks: "0001"
`,
				"defaultNodepoolConfig": `
machineType: "tpu7x-standard-4t"
topology: "2x2"
placementPolicy: "tpu-provisioner-2x2"
nodeCount: 1
`,
			},
			wantErr:         true,
			wantErrContains: "must be a 3D shape like '2x2x1'",
		},
		{
			name: "Rejected config with unsupported 3D topology",
			cmData: map[string]string{
				"reservations": `
- name: "my-gsc-res"
  gscSubblocks:
    - block: "block-1"
      subblocks: "0001"
      nodepoolConfig:
        machineType: "tpu7x-standard-4t"
        topology: "3x3x3"
        placementPolicy: "tpu-provisioner-3x3x3"
        nodeCount: 1
`,
			},
			wantErr:         true,
			wantErrContains: "is not supported by this provisioner",
		},
		{
			name: "Mismatch nodeCount and expected nodeCount",
			cmData: map[string]string{
				"reservations": `
- name: "my-gsc-res"
  gscSubblocks:
    - block: "block-1"
      subblocks: "0001"
`,
				"defaultNodepoolConfig": `
machineType: "tpu7x-standard-4t"
topology: "4x4x4"
placementPolicy: "tpu-provisioner-4x4x4"
nodeCount: 8
`,
			},
			wantErr:         true,
			wantErrContains: "requires exactly 16 nodes, but nodeCount is set to 8",
		},
		{
			name: "Rejected config with non-v7x machine type",
			cmData: map[string]string{
				"reservations": `
- name: "my-gsc-res"
  gscSubblocks:
    - block: "block-1"
      subblocks: "0001"
      nodepoolConfig:
        machineType: "tpu-v5e-lite-1t"
        topology: "2x2x1"
        placementPolicy: "tpu-provisioner-2x2x1"
        nodeCount: 4
`,
			},
			wantErr:         true,
			wantErrContains: "strictly supports 'tpu7x-standard-4t'",
		},
		{
			name: "Rejected topology larger than 4x4x4",
			cmData: map[string]string{
				"reservations": `
- name: "my-gsc-res"
  gscSubblocks:
    - block: "block-1"
      subblocks: "0001"
      nodepoolConfig:
        machineType: "tpu7x-standard-4t"
        topology: "4x4x8"
        placementPolicy: "tpu-provisioner-4x4x8"
        nodeCount: 32
`,
			},
			wantErr:         true,
			wantErrContains: "topology \"4x4x8\" is not supported by this provisioner. Supported topologies are: 2x2x1, 2x2x2, 2x2x4, 2x4x4, 4x4x4",
		},
		{
			name: "Rejected config with empty gscSubblocks list",
			cmData: map[string]string{
				"reservations": `
- name: "my-gsc-res"
  gscSubblocks: []
`,
				"defaultNodepoolConfig": defaultNodepoolConfig4x4x4,
			},
			wantErr:         true,
			wantErrContains: "gscSubblocks list cannot be empty",
		},
		{
			name: "Rejected config with empty subblocks inside a subblock",
			cmData: map[string]string{
				"reservations": `
- name: "my-gsc-res"
  gscSubblocks:
    - block: "block-1"
      subblocks: ""
`,
				"defaultNodepoolConfig": defaultNodepoolConfig4x4x4,
			},
			wantErr:         true,
			wantErrContains: "subblocks cannot be empty",
		},
		{
			name: "Rejected config with empty placementPolicy",
			cmData: map[string]string{
				"reservations": `
- name: "my-gsc-res"
  gscSubblocks:
    - block: "block-1"
      subblocks: "0001"
`,
				"defaultNodepoolConfig": `
machineType: "tpu7x-standard-4t"
topology: "2x2x1"
placementPolicy: ""
nodeCount: 1
`,
			},
			wantErr:         true,
			wantErrContains: "placementPolicy cannot be empty",
		},
		{
			name: "Valid config with empty reservations list",
			cmData: map[string]string{
				"reservations":          `[]`,
				"defaultNodepoolConfig": defaultNodepoolConfig4x4x4,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tpu-provisioner-static-nodepools-config",
					Namespace: "tpu-provisioner-system",
				},
				Data: tt.cmData,
			}

			err := ValidateConfigMap(cm)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ValidateConfigMap() succeeded; want error containing %q", tt.wantErrContains)
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("ValidateConfigMap() error = %v, want error containing %q", err, tt.wantErrContains)
				}
			} else {
				if err != nil {
					t.Fatalf("ValidateConfigMap() returned unexpected error: %v", err)
				}
			}
		})
	}
}
