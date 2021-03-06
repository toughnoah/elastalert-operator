package e2e

import (
	"bytes"
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/toughnoah/elastalert-operator/api/v1alpha1"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
		"alert": []string{
			"post",
		},
		"http_post_url":     "https://test.com/alerts",
		"http_post_timeout": 60,
	}
	RuleSample2 = map[string]interface{}{
		"name":  "check-elastalert",
		"type":  "any",
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
		"alert": []string{
			"post",
		},
		"http_post_url":     "https://test.com/alerts",
		"http_post_timeout": 60,
	}
	Key = types.NamespacedName{
		Name:      "e2e-elastalert",
		Namespace: "default",
	}
)

var _ = Describe("Elastalert Controller", func() {
	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		ea := &v1alpha1.Elastalert{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: Key.Namespace,
				Name:      Key.Name,
			},
		}
		_ = k8sClient.Delete(context.Background(), ea)
	})

	Context("Deploy Elastalert", func() {
		It("Test create Elastalert with wrong config", func() {
			elastalert := &v1alpha1.Elastalert{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: Key.Namespace,
					Name:      Key.Name,
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

			By("Check the cert secret exists.")
			Eventually(func() error {
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      "e2e-elastalert-es-cert",
					Namespace: "default",
				}, &v1.Secret{})
				return err
			}, timeout, interval).Should(Succeed())

			elastalert = &v1alpha1.Elastalert{}

			By("Start checking for INITIALIZING status")
			Eventually(func() string {
				_ = k8sClient.Get(context.Background(), Key, elastalert)
				return elastalert.Status.Phase
			}, timeout*8, interval).Should(Equal("INITIALIZING"))

			By("Start waiting for FAILED status with wrong config")
			Eventually(func() string {
				_ = k8sClient.Get(context.Background(), Key, elastalert)
				return elastalert.Status.Phase
			}, timeout*8, interval).Should(Equal("FAILED"))

			By("Update elastalert config.yaml then check restart.")
			Expect(k8sClient.Get(context.Background(), Key, elastalert)).To(Succeed())

			By("Start update elastalert pod template and fix the wrong config")
			elastalert.ObjectMeta.Annotations = map[string]string{
				"sidecar.istio.io/inject": "false",
			}
			elastalert.Spec.PodTemplateSpec.Spec.Containers = append(elastalert.Spec.PodTemplateSpec.Spec.Containers, v1.Container{
				Name: "elastalert",
				Resources: v1.ResourceRequirements{
					Limits: map[v1.ResourceName]resource.Quantity{
						v1.ResourceMemory: resource.MustParse("4Gi"),
						v1.ResourceCPU:    resource.MustParse("2"),
					},
					Requests: map[v1.ResourceName]resource.Quantity{
						v1.ResourceMemory: resource.MustParse("1Gi"),
						v1.ResourceCPU:    resource.MustParse("1"),
					},
				},
			})
			elastalert.Spec.ConfigSetting = v1alpha1.NewFreeForm(ConfigSample)
			elastalert.Spec.Rule = []v1alpha1.FreeForm{
				v1alpha1.NewFreeForm(RuleSample1),
				v1alpha1.NewFreeForm(RuleSample2),
			}
			Expect(k8sClient.Update(context.Background(), elastalert)).To(Succeed())

			By("Start checking for INITIALIZING status again")
			Eventually(func() string {
				_ = k8sClient.Get(context.Background(), Key, elastalert)
				return elastalert.Status.Phase
			}, timeout*8, interval).Should(Equal("INITIALIZING"))

			By("Check RUNNING status")
			Eventually(func() string {
				_ = k8sClient.Get(context.Background(), Key, elastalert)
				return elastalert.Status.Phase
			}, timeout*8, interval).Should(Equal("RUNNING"))

			By("Start waiting deployment to be stable.")
			dep := &appsv1.Deployment{}
			Eventually(func() int {
				_ = k8sClient.Get(context.Background(), Key, dep)
				return int(dep.Status.AvailableReplicas)
			}, timeout*4, interval).Should(Equal(1))

			By("Check pod resources")
			Eventually(func() bool {
				_ = k8sClient.Get(context.Background(), Key, dep)
				if len(dep.Spec.Template.Spec.Containers) == 0 {
					return false
				}
				return reflect.DeepEqual(dep.Spec.Template.Spec.Containers[0].Resources, elastalert.Spec.PodTemplateSpec.Spec.Containers[0].Resources)
			}, timeout, interval).Should(Equal(true))

			By("Check pod annotations")
			Expect(dep.Spec.Template.Annotations["sidecar.istio.io/inject"]).Should(Equal("false"))

			By("check elastalert rules.")
			Eventually(func() bool {
				RuleConfigMap := &v1.ConfigMap{}
				_ = k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      "e2e-elastalert-rule",
					Namespace: "default",
				}, RuleConfigMap)
				return compare(RuleConfigMap.Data["test-elastalert.yaml"], RuleSample1) && compare(RuleConfigMap.Data["check-elastalert.yaml"], RuleSample2)
			}, timeout, interval).Should(Equal(true))

			By("Check config.yaml configmap.")
			Eventually(func() bool {
				configConfigMap := &v1.ConfigMap{}
				_ = k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      "e2e-elastalert-config",
					Namespace: "default",
				}, configConfigMap)
				return compare(configConfigMap.Data["config.yaml"], ConfigSample)
			}, timeout, interval).Should(Equal(true))

			By("Start to test deployment reconcile")
			Eventually(func() error {
				dep = &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      Key.Name,
						Namespace: Key.Namespace,
					},
				}
				return k8sClient.Delete(context.Background(), dep)
			}, timeout, interval).Should(Succeed())

			By("Start waiting deployment to be stable.")
			Eventually(func() int {
				dep = &appsv1.Deployment{}
				_ = k8sClient.Get(context.Background(), Key, dep)
				return int(dep.Status.AvailableReplicas)
			}, timeout*4, interval).Should(Equal(1))

		})
	})
})

func compare(source string, dest map[string]interface{}) bool {
	out, _ := yaml.Marshal(dest)
	return bytes.Compare([]byte(source), out) == 0
}
