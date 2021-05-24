package controllers

import (
	"context"
	"elastalert/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

var _ = Describe("Elastalert Controller", func() {
	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})
	Context("Deploy Elastalert", func() {
		It("test creat Elastalert", func() {
			key := types.NamespacedName{
				Name:      "e2e-elastalert",
				Namespace: "default",
			}
			elastalert := &v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: key.Name,
					Name:      key.Namespace,
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
				},
			}

			Expect(k8sClient.Create(context.Background(), elastalert)).ShouldNot(Succeed())
			By("go here, test success ")
			time.Sleep(time.Second * 30)
		})
	})
})
