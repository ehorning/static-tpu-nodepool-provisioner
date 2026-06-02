package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestConfigMapValidator_Handle(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	decoder := admission.NewDecoder(scheme)

	v := &ConfigMapValidator{}
	v.InjectDecoder(decoder)

	tests := []struct {
		name         string
		cmName       string
		cmData       map[string]string
		wantAllowed  bool
		wantHTTPCode int32
	}{
		{
			name:   "Ignore other ConfigMaps",
			cmName: "some-other-configmap",
			cmData: map[string]string{
				"reservations": "invalid stuff",
			},
			wantAllowed:  true,
			wantHTTPCode: http.StatusOK,
		},
		{
			name:   "Valid ConfigMap",
			cmName: ConfigMapName,
			cmData: map[string]string{
				"defaultNodepoolConfig": `
machineType: "tpu7x-standard-4t"
topology: "4x4x4"
placementPolicy: "tpu-provisioner-4x4x4"
nodeCount: 16
`,
				"reservations": `
- name: "my-gsc-res"
  gscSubblocks:
    - block: "block-1"
      subblocks: "0001"
`,
			},
			wantAllowed:  true,
			wantHTTPCode: http.StatusOK,
		},
		{
			name:   "Invalid ConfigMap is rejected",
			cmName: ConfigMapName,
			cmData: map[string]string{
				"defaultNodepoolConfig": `
machineType: "tpu7x-standard-4t"
topology: "4x4x4"
placementPolicy: "tpu-provisioner-4x4x4"
nodeCount: 1
`,
				"reservations": `
- name: "my-gsc-res"
  gscSubblocks:
    - block: "block-1"
      subblocks: "0001"
`,
			},
			wantAllowed:  false,
			wantHTTPCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.cmName,
					Namespace: "tpu-provisioner-system",
				},
				Data: tt.cmData,
			}

			raw, err := json.Marshal(cm)
			if err != nil {
				t.Fatalf("Failed to marshal configmap: %v", err)
			}

			req := admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{},
			}
			req.Object.Raw = raw

			resp := v.Handle(context.Background(), req)

			if resp.Allowed != tt.wantAllowed {
				t.Errorf("Handle() Allowed = %v, want %v", resp.Allowed, tt.wantAllowed)
			}
			if resp.Result.Code != tt.wantHTTPCode {
				t.Errorf("Handle() HTTP Code = %v, want %v", resp.Result.Code, tt.wantHTTPCode)
			}
		})
	}
}
