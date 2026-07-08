package webhooktest

import (
	"context"
	"time"

	"github.com/GoogleCloudPlatform/ai-on-gke/static-np-provisioner/internal/controller"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ConfigMap Validation Webhook", func() {
	Context("when creating a ConfigMap", func() {
		AfterEach(func() {
			ctx := context.Background()
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      controller.ConfigMapName,
					Namespace: "tpu-provisioner-system",
				},
			}
			_ = k8sClient.Delete(ctx, cm)
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKeyFromObject(cm), cm)
			}, time.Second*5, time.Millisecond*100).ShouldNot(Succeed())
		})

		It("should allow a valid ConfigMap to be created", func() {
			ctx := context.Background()

			// Ensure namespace exists
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "tpu-provisioner-system"}}
			_ = k8sClient.Create(ctx, ns)

			validCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      controller.ConfigMapName,
					Namespace: "tpu-provisioner-system",
					Labels: map[string]string{
						"tpu-provisioner-managed": "true",
					},
				},
				Data: map[string]string{
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
			}

			err := k8sClient.Create(ctx, validCm)
			Expect(err).NotTo(HaveOccurred(), "valid configmap should be admitted by webhook")
		})

		It("should reject an invalid ConfigMap via the webhook", func() {
			ctx := context.Background()

			// Ensure namespace exists
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "tpu-provisioner-system"}}
			_ = k8sClient.Create(ctx, ns)

			invalidCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      controller.ConfigMapName,
					Namespace: "tpu-provisioner-system",
					Labels: map[string]string{
						"tpu-provisioner-managed": "true",
					},
				},
				Data: map[string]string{
					"defaultNodepoolConfig": `
machineType: "tpu7x-standard-4t"
topology: "4x4x4"
placementPolicy: "tpu-provisioner-4x4x4"
nodeCount: 9999
`,
					"reservations": `
- name: "my-gsc-res"
  gscSubblocks:
    - block: "block-1"
      subblocks: "0001"
`,
				},
			}

			err := k8sClient.Create(ctx, invalidCm)
			Expect(err).To(HaveOccurred(), "invalid configmap should be rejected by webhook")
			Expect(err.Error()).To(ContainSubstring("requires exactly 16 nodes, but nodeCount is set to 9999"))
		})
	})
})
