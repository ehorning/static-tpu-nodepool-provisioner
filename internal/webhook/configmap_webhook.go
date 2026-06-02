package webhook

import (
	"context"
	"net/http"

	"github.com/GoogleCloudPlatform/ai-on-gke/static-np-provisioner/internal/config"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	ConfigMapName = "tpu-provisioner-static-nodepools-config"
)

// ConfigMapValidator validates static nodepool configurations in ConfigMaps.
type ConfigMapValidator struct {
	Client  client.Client
	decoder admission.Decoder
}

// Handle intercepts admission requests for ConfigMaps and validates the custom static TPU configurations.
func (v *ConfigMapValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	cm := &corev1.ConfigMap{}

	// Decode the ConfigMap object from the admission request.
	if err := v.decoder.Decode(req, cm); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Only intercept and validate our dedicated static nodepool configmap.
	if cm.Name != ConfigMapName {
		return admission.Allowed("ConfigMap is not managed by static-np-provisioner")
	}

	// Execute high-fidelity topology and configuration checks.
	if err := config.ValidateConfigMap(cm); err != nil {
		return admission.Denied(err.Error())
	}

	return admission.Allowed("Configuration is valid")
}

// InjectDecoder injects the decoder.
func (v *ConfigMapValidator) InjectDecoder(d admission.Decoder) error {
	v.decoder = d
	return nil
}
