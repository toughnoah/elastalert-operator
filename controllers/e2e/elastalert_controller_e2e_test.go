package e2e

import (
	"context"
	"elastalert/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

const timeout = time.Minute * 4
const interval = time.Second * 1

var _ = Describe("Elastalert Controller", func() {
	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		key := types.NamespacedName{
			Name:      "e2e-elastalert",
			Namespace: "default",
		}
		ea := &v1alpha1.Elastalert{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: key.Namespace,
				Name:      key.Name,
			},
		}
		_ = k8sClient.Delete(context.Background(), ea)
		// Add any teardown steps that needs to be executed after each test
	})
	Context("Deploy Elastalert", func() {
		It("Test creat Elastalert with wrong config", func() {
			key := types.NamespacedName{
				Name:      "e2e-elastalert",
				Namespace: "default",
			}
			elastalert := &v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: key.Namespace,
					Name:      key.Name,
				},
				Spec: v1alpha1.ElastalertSpec{
					ConfigSetting: v1alpha1.NewFreeForm(map[string]interface{}{
						"config": "test",
					}),
					Rule: []v1alpha1.FreeForm{
						v1alpha1.NewFreeForm(map[string]interface{}{
							"name": "test-elastalert", "type": "any",
						}),
					},
					PodTemplateSpec: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			}
			Expect(k8sClient.Create(context.Background(), elastalert)).Should(Succeed())
			By("Start waiting for failed status")
			Eventually(func() string {
				ea := &v1alpha1.Elastalert{}
				_ = k8sClient.Get(context.Background(), key, ea)
				return ea.Status.Phase
			}, timeout, interval).Should(Equal("FAILED"))
		})
	})
})
