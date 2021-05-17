// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package podspec

import (
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
)

// Defaulter ensures that values are set if none exists in the base container.
type Defaulter struct {
	base *corev1.Container
}

// Container returns a copy of the resulting container.
func (d Defaulter) Container() corev1.Container {
	return *d.base.DeepCopy()
}

func NewDefaulter(base *corev1.Container) Defaulter {
	return Defaulter{
		base: base,
	}
}

// From inherits default values from an other container.
func (d Defaulter) From(other corev1.Container) Defaulter {
	if other.Lifecycle != nil {
		d.WithPreStopHook(other.Lifecycle.PreStop)
	}

	return d.
		WithImage(other.Image).
		WithCommand(other.Command).
		WithArgs(other.Args).
		WithPorts(other.Ports).
		WithEnv(other.Env).
		WithResources(other.Resources).
		WithVolumeMounts(other.VolumeMounts).
		WithReadinessProbe(other.ReadinessProbe)
}

func (d Defaulter) WithCommand(command []string) Defaulter {
	if len(d.base.Command) == 0 {
		d.base.Command = command
	}
	return d
}

func (d Defaulter) WithArgs(args []string) Defaulter {
	if len(d.base.Args) == 0 {
		d.base.Args = args
	}
	return d
}

func (d Defaulter) WithPorts(ports []corev1.ContainerPort) Defaulter {
	for _, p := range ports {
		if !d.portExists(p.Name) {
			d.base.Ports = append(d.base.Ports, p)
		}
	}
	// order ports by name to ensure stable pod spec comparison
	sort.SliceStable(d.base.Ports, func(i, j int) bool {
		return d.base.Ports[i].Name < d.base.Ports[j].Name
	})
	return d
}

// portExists checks if a port with the given name already exists in the Container.
func (d Defaulter) portExists(name string) bool {
	for _, p := range d.base.Ports {
		if p.Name == name {
			return true
		}
	}
	return false
}

// WithImage sets up the Container Docker image, unless already provided.
// The default image will be used unless customImage is not empty.
func (d Defaulter) WithImage(image string) Defaulter {
	if d.base.Image == "" {
		d.base.Image = image
	}
	return d
}

func (d Defaulter) WithReadinessProbe(readinessProbe *corev1.Probe) Defaulter {
	if d.base.ReadinessProbe == nil {
		d.base.ReadinessProbe = readinessProbe
	}
	return d
}

// envExists checks if an env var with the given name already exists in the provided slice.
func (d Defaulter) envExists(name string) bool {
	for _, v := range d.base.Env {
		if v.Name == name {
			return true
		}
	}
	return false
}

func (d Defaulter) WithEnv(vars []corev1.EnvVar) Defaulter {
	for _, v := range vars {
		if !d.envExists(v.Name) {
			d.base.Env = append(d.base.Env, v)
		}
	}
	return d
}

// WithResources ensures that resource requirements are set in the container.
func (d Defaulter) WithResources(resources corev1.ResourceRequirements) Defaulter {
	if d.base.Resources.Requests == nil && d.base.Resources.Limits == nil {
		d.base.Resources = resources
	}
	return d
}

// volumeExists checks if a volume mount with the given name already exists in the Container.
func (d Defaulter) volumeMountExists(volumeMount corev1.VolumeMount) bool {
	for _, v := range d.base.VolumeMounts {
		if v.Name == volumeMount.Name || v.MountPath == volumeMount.MountPath {
			return true
		}
	}
	return false
}

func (d Defaulter) WithVolumeMounts(volumeMounts []corev1.VolumeMount) Defaulter {
	for _, v := range volumeMounts {
		if !d.volumeMountExists(v) {
			d.base.VolumeMounts = append(d.base.VolumeMounts, v)
		}
	}
	// order volume mounts by name to ensure stable pod spec comparison
	sort.SliceStable(d.base.VolumeMounts, func(i, j int) bool {
		return d.base.VolumeMounts[i].Name < d.base.VolumeMounts[j].Name
	})
	return d
}

func (d Defaulter) WithPreStopHook(handler *corev1.Handler) Defaulter {
	if d.base.Lifecycle == nil {
		d.base.Lifecycle = &corev1.Lifecycle{}
	}

	if d.base.Lifecycle.PreStop == nil {
		// no user-provided hook, we can use our own
		d.base.Lifecycle.PreStop = handler
	}

	return d
}

// WithLabels sets the given labels, but does not override those that already exist.
func (b *PodTemplateBuilder) WithLabels(labels map[string]string) *PodTemplateBuilder {
	b.PodTemplate.Labels = MergePreservingExistingKeys(b.PodTemplate.Labels, labels)
	return b
}

// WithAnnotations sets the given annotations, but does not override those that already exist.
func (b *PodTemplateBuilder) WithAnnotations(annotations map[string]string) *PodTemplateBuilder {
	b.PodTemplate.Annotations = MergePreservingExistingKeys(b.PodTemplate.Annotations, annotations)
	return b
}

// WithDockerImage sets up the Container Docker image, unless already provided.
// The default image will be used unless customImage is not empty.
func (b *PodTemplateBuilder) WithDockerImage(customImage string, defaultImage string) *PodTemplateBuilder {
	if customImage != "" {
		b.containerDefaulter.WithImage(customImage)
	} else {
		b.containerDefaulter.WithImage(defaultImage)
	}
	return b
}

// WithReadinessProbe sets up the given readiness probe, unless already provided in the template.
func (b *PodTemplateBuilder) WithReadinessProbe(readinessProbe corev1.Probe) *PodTemplateBuilder {
	b.containerDefaulter.WithReadinessProbe(&readinessProbe)
	return b
}

// WithAffinity sets a default affinity, unless already provided in the template.
// An empty affinity in the spec is not overridden.
func (b *PodTemplateBuilder) WithAffinity(affinity *corev1.Affinity) *PodTemplateBuilder {
	if b.PodTemplate.Spec.Affinity == nil {
		b.PodTemplate.Spec.Affinity = affinity
	}
	return b
}

// WithPorts appends the given ports to the Container ports, unless already provided in the template.
func (b *PodTemplateBuilder) WithPorts(ports []corev1.ContainerPort) *PodTemplateBuilder {
	b.containerDefaulter.WithPorts(ports)
	return b
}

// WithCommand sets the given command to the Container, unless already provided in the template.
func (b *PodTemplateBuilder) WithCommand(command []string) *PodTemplateBuilder {
	b.containerDefaulter.WithCommand(command)
	return b
}

// volumeExists checks if a volume with the given name already exists in the Container.
func (b *PodTemplateBuilder) volumeExists(name string) bool {
	for _, v := range b.PodTemplate.Spec.Volumes {
		if v.Name == name {
			return true
		}
	}
	return false
}

// WithVolumes appends the given volumes to the Container, unless already provided in the template.
func (b *PodTemplateBuilder) WithVolumes(volumes ...corev1.Volume) *PodTemplateBuilder {
	for _, v := range volumes {
		if !b.volumeExists(v.Name) {
			b.PodTemplate.Spec.Volumes = append(b.PodTemplate.Spec.Volumes, v)
		}
	}
	// order volumes by name to ensure stable pod spec comparison
	sort.SliceStable(b.PodTemplate.Spec.Volumes, func(i, j int) bool {
		return b.PodTemplate.Spec.Volumes[i].Name < b.PodTemplate.Spec.Volumes[j].Name
	})
	return b
}

// WithVolumeMounts appends the given volume mounts to the Container, unless already provided in the template.
func (b *PodTemplateBuilder) WithVolumeMounts(volumeMounts ...corev1.VolumeMount) *PodTemplateBuilder {
	b.containerDefaulter.WithVolumeMounts(volumeMounts)
	return b
}

// WithEnv appends the given env vars to the Container, unless already provided in the template.
func (b *PodTemplateBuilder) WithEnv(vars ...corev1.EnvVar) *PodTemplateBuilder {
	b.containerDefaulter.WithEnv(vars)
	return b
}

// WithTerminationGracePeriod sets the given termination grace period if not already specified in the template.
func (b *PodTemplateBuilder) WithTerminationGracePeriod(period int64) *PodTemplateBuilder {
	if b.PodTemplate.Spec.TerminationGracePeriodSeconds == nil {
		b.PodTemplate.Spec.TerminationGracePeriodSeconds = &period
	}
	return b
}

// WithInitContainerDefaults sets default values for the current init containers.
//
// Defaults:
// - If the init container contains an empty image field, it's inherited from the elastalert container.
// - VolumeMounts from the elastalert container are added to the init container VolumeMounts, unless they would conflict
//   with a specified VolumeMount (by having the same VolumeMount.Name or VolumeMount.MountPath)
// - default environment variables
//
// This method can also be used to set some additional environment variables.
func (b *PodTemplateBuilder) WithInitContainerDefaults(additionalEnvVars ...corev1.EnvVar) *PodTemplateBuilder {
	elastalertContainer := b.containerDefaulter.Container()
	for i := range b.PodTemplate.Spec.InitContainers {
		b.PodTemplate.Spec.InitContainers[i] =
			NewDefaulter(&b.PodTemplate.Spec.InitContainers[i]).
				// Inherit image and volume mounts from elastalert container in the Pod
				WithImage(elastalertContainer.Image).
				WithVolumeMounts(elastalertContainer.VolumeMounts).
				Container()
	}
	return b
}

// findInitContainerByName attempts to find an init container with the given name in the template
// Returns the index of the container or -1 if no init container by that name was found.
func (b *PodTemplateBuilder) findInitContainerByName(name string) int {
	for i, c := range b.PodTemplate.Spec.InitContainers {
		if c.Name == name {
			return i
		}
	}
	return -1
}

// WithInitContainers includes the given init containers to the pod template.
//
// Ordering:
// - Provided init containers are prepended to the existing ones in the template.
// - If an init container by the same name already exists in the template, the two PodTemplates are merged, the values
// provided by the user take precedence.
func (b *PodTemplateBuilder) WithInitContainers(
	initContainers ...corev1.Container,
) *PodTemplateBuilder {
	var containers []corev1.Container

	for _, c := range initContainers {
		if index := b.findInitContainerByName(c.Name); index != -1 {
			userContainer := b.PodTemplate.Spec.InitContainers[index]

			// remove it from the podTemplate
			b.PodTemplate.Spec.InitContainers = append(
				b.PodTemplate.Spec.InitContainers[:index],
				b.PodTemplate.Spec.InitContainers[index+1:]...,
			)

			// Create a container based on what the user specified but ensure that values
			// are set if none are provided.
			containers = append(containers,
				// Set the container provided by the user as the base.
				NewDefaulter(userContainer.DeepCopy()).
					// Inherit all other values from the container built by the controller.
					From(c).
					Container())
			fmt.Println(NewDefaulter(userContainer.DeepCopy()).
				// Inherit all other values from the container built by the controller.
				From(c).
				Container())
		} else {
			containers = append(containers, c)
		}
	}
	b.PodTemplate.Spec.InitContainers = append(containers, b.PodTemplate.Spec.InitContainers...)
	return b
}

// WithResources sets up the given resource requirements if both resources limits and requests
// are nil in the main container.
// If a zero-value (empty map) for at least one of limits or request is provided, the given resource requirements
// are not applied: the user may want to use a LimitRange.
func (b *PodTemplateBuilder) WithResources(resources corev1.ResourceRequirements) *PodTemplateBuilder {
	b.containerDefaulter.WithResources(resources)
	return b
}

func (b *PodTemplateBuilder) WithPreStopHook(handler corev1.Handler) *PodTemplateBuilder {
	b.containerDefaulter.WithPreStopHook(&handler)
	return b
}

func (b *PodTemplateBuilder) WithArgs(args ...string) *PodTemplateBuilder {
	b.containerDefaulter.WithArgs(args)
	return b
}

func (b *PodTemplateBuilder) WithServiceAccount(serviceAccount string) *PodTemplateBuilder {
	if b.PodTemplate.Spec.ServiceAccountName == "" {
		b.PodTemplate.Spec.ServiceAccountName = serviceAccount
	}
	return b
}

func (b *PodTemplateBuilder) WithHostNetwork() *PodTemplateBuilder {
	b.PodTemplate.Spec.HostNetwork = true
	return b
}

func (b *PodTemplateBuilder) WithDNSPolicy(dnsPolicy corev1.DNSPolicy) *PodTemplateBuilder {
	if b.PodTemplate.Spec.DNSPolicy == "" {
		b.PodTemplate.Spec.DNSPolicy = dnsPolicy
	}
	return b
}

func (b *PodTemplateBuilder) WithPodSecurityContext(securityContext corev1.PodSecurityContext) *PodTemplateBuilder {
	if b.PodTemplate.Spec.SecurityContext == nil {
		b.PodTemplate.Spec.SecurityContext = &securityContext
	}
	return b
}

func (b *PodTemplateBuilder) WithAutomountServiceAccountToken() *PodTemplateBuilder {
	if b.PodTemplate.Spec.AutomountServiceAccountToken == nil {
		t := true
		b.PodTemplate.Spec.AutomountServiceAccountToken = &t
	}
	return b
}

func NewPreStopHook() *corev1.Handler {
	return &corev1.Handler{}
}
