// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package podspec

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	// DefaultTerminationGracePeriodSeconds is the termination grace period for the Elasalert containers
	DefaultTerminationGracePeriodSeconds int64 = 30
	DefaultElastAlertName                      = "elastalert"
	DefautlImage                               = "toughnoah/elastalert:v1.0"
	DefaultCertVolumeName                      = "elasticsearch-cert"
	DefaultCertSuffix                          = "-es-cert"
	DefaultCertMountPath                       = "/ssl"
	DefaultElasticCertName                     = "elasticCA.crt"
	DefaultRulesFolder                         = "/etc/elastalert/rules/..data/"
	DefaultElasticCertPath                     = "/ssl/elasticCA.crt"
)

var (
	DefaultMemoryLimits = resource.MustParse("2Gi")
	// DefaultResources for the Elasalert container. The JVM default heap size is 1Gi, so we
	// request at least 2Gi for the container to make sure ES can work properly.
	// Not applying this minimum default would make Elasalert randomly crash (OOM) on small machines.
	// Similarly, we apply a default memory limit of 2Gi, to ensure the Pod isn't the first one to get evicted.
	// No CPU requirement is set by default.
	DefaultResources = corev1.ResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceMemory: DefaultMemoryLimits,
		},
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceMemory: DefaultMemoryLimits,
		},
	}
)

// DefaultAffinity returns the default affinity for pods in a cluster.
func DefaultAffinity(esName string) *corev1.Affinity {
	return &corev1.Affinity{
		// prefer to avoid two pods in the same cluster being co-located on a single node
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{},
		},
	}
}
