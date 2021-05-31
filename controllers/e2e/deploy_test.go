package e2e

import (
	"context"
	"elastalert/api/v1alpha1"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"time"
)

const interval = time.Second * 1
const timeout = time.Second * 30

var (
	ConfigSample = map[string]interface{}{
		"es_host":         "es.com.cn",
		"es_port":         9200,
		"use_ssl":         true,
		"es_username":     "elastic",
		"es_password":     "changeme",
		"verify_certs":    false,
		"writeback_index": "elastalert",
		"rules_folder":    "/etc/elastalert/rules/..data/",
		"run_every": map[string]interface{}{
			"minutes": 1,
		},
		"buffer_time": map[string]interface{}{
			"minutes": 15,
		},
	}
	RuleSample1 = map[string]interface{}{
		"name":  "test-elastalert",
		"type":  "any",
		"index": "api-*",
		"filter": []map[string]interface{}{
			{
				"query": map[string]interface{}{
					"query_string": map[string]interface{}{
						"query": "http_status_code: 503",
					},
				},
			},
		},
	}
	RuleSample2 = map[string]interface{}{
		"name":  "check-elastalert",
		"type":  "aggs",
		"index": "kpi-*",
		"filter": []map[string]interface{}{
			{
				"query": map[string]interface{}{
					"query_string": map[string]interface{}{
						"query": "http_status_code: 600",
					},
				},
			},
		},
	}
)

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
		It("Test create Elastalert", func() {
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
					ConfigSetting: v1alpha1.NewFreeForm(ConfigSample),
					Rule: []v1alpha1.FreeForm{
						v1alpha1.NewFreeForm(RuleSample1),
						v1alpha1.NewFreeForm(RuleSample2),
					},
					Alert: v1alpha1.NewFreeForm(map[string]interface{}{
						"alert": []string{
							"post",
						},
						"http_post_url":     "https://test.com/alerts",
						"http_post_timeout": 60,
					}),
					PodTemplateSpec: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"e2e": "test",
							},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{},
						},
					},
				},
			}
			By("Start to deploy elastalert.")
			Expect(k8sClient.Create(context.Background(), elastalert)).Should(Succeed())

			By("Check config.yaml configmap.")
			Eventually(func() bool {
				configConfigMap := &v1.ConfigMap{}
				_ = k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      "e2e-elastalert-config",
					Namespace: "default",
				}, configConfigMap)
				return compare(configConfigMap.Data["config.yaml"], ConfigSample)
			}, timeout, interval).Should(Equal(true))

			By("Check rules configmap.")
			Eventually(func() bool {
				RuleConfigMap := &v1.ConfigMap{}
				_ = k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      "e2e-elastalert-rule",
					Namespace: "default",
				}, RuleConfigMap)
				return compare(RuleConfigMap.Data["test-elastalert.yaml"], RuleSample1) && compare(RuleConfigMap.Data["check-elastalert.yaml"], RuleSample2)
			}, timeout, interval).Should(Equal(true))

			By("Check cert secret.")
			Eventually(func() error {
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      "e2e-elastalert-es-cert",
					Namespace: "default",
				}, &v1.Secret{})
				return err
			}, timeout, interval).Should(Succeed())

			By("Start waiting deployment to be stable.")
			Eventually(func() int32 {
				dep := &appsv1.Deployment{}
				_ = k8sClient.Get(context.Background(), key, dep)
				return dep.Status.AvailableReplicas
			}, timeout*2, interval).Should(Equal(1))
		})
		It("Test create Elastalert with wrong config", func() {
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
							"name": "test-elastalert",
							"type": "any",
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
			}, timeout*6, interval).Should(Equal("FAILED"))
		})
	})
})

func compare(source string, dest map[string]interface{}) bool {
	out, _ := yaml.Marshal(dest)
	fmt.Println(reflect.DeepEqual([]byte(source), out))
	return reflect.DeepEqual([]byte(source), out)
}
