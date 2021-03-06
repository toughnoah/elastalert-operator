package observer

import (
	"context"
	"fmt"
	esv1alpha1 "github.com/toughnoah/elastalert-operator/api/v1alpha1"
	"github.com/toughnoah/elastalert-operator/controllers/event"
	"github.com/toughnoah/elastalert-operator/controllers/podspec"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"time"
)

const name = "observation"

var log = ctrl.Log.WithName(name)

// Observer regularly check the health of elastalert deployment
// in a thread-safe way
type Observer struct {
	elastalert          types.NamespacedName
	creationTime        time.Time
	stopChan            chan struct{}
	stopOnce            sync.Once
	mutex               sync.RWMutex
	ObservationInterval time.Duration
	client              client.Client
	recorder            record.EventRecorder
}

// NewObserver creates and starts an Observer
func NewObserver(c client.Client, elastalert types.NamespacedName, interval time.Duration, recorder record.EventRecorder) *Observer {
	observer := Observer{
		elastalert:          elastalert,
		client:              c,
		creationTime:        time.Now(),
		stopChan:            make(chan struct{}),
		stopOnce:            sync.Once{},
		ObservationInterval: interval,
		recorder:            recorder,
	}
	return &observer
}

// Start the observer in a separate goroutine
func (o *Observer) Start() {
	log.Info(
		"Starting observer for elastalert instance.",
		"namespace", o.elastalert.Namespace,
		"elastalert", o.elastalert.Name,
	)
	go o.runPeriodically()
}

// Stop the observer loop
func (o *Observer) Stop() {
	log.Info(
		"Stopping observer for deleted elastalert instance.",
		"namespace", o.elastalert.Namespace,
		"elastalert", o.elastalert.Name,
	)
	o.stopOnce.Do(func() {
		close(o.stopChan)
	})
}

func (o *Observer) runPeriodically() {
	ticker := time.NewTicker(o.ObservationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			o.checkDeploymentHeath()
		case <-o.stopChan:
			return
		}
	}
}
func (o *Observer) checkDeploymentHeath() error {
	ea := &esv1alpha1.Elastalert{}
	err := o.client.Get(context.Background(), o.elastalert, ea)
	if err != nil {
		log.Error(err, "Failed to get elastalert instance while observing.", "namespace", o.elastalert.Namespace, "elastalert", o.elastalert.Name)
		return err
	}
	dep := &appsv1.Deployment{}
	err = o.client.Get(context.Background(), o.elastalert, dep)
	if err != nil {
		log.Error(err, "Failed to get deployment instance while observing.", "namespace", o.elastalert.Namespace, "elastalert", o.elastalert.Name)
		EmitK8sEvent(o.recorder, ea, corev1.EventTypeWarning, event.EventReasonError, "Get deployment instance failed while observing.")
		return UpdateElastalertStatus(o.client, context.Background(), ea, esv1alpha1.ActionFailed)
	}
	if dep.Status.AvailableReplicas != *dep.Spec.Replicas {
		log.Error(err, "AvailableReplicas of deployment instance is 0 .", "namespace", o.elastalert.Namespace, "elastalert", o.elastalert.Name)
		EmitK8sEvent(o.recorder, ea, corev1.EventTypeWarning, event.EventReasonError, "AvailableReplicas of deployment instance is 0.")
		return UpdateElastalertStatus(o.client, context.Background(), ea, esv1alpha1.ActionFailed)
	}
	if dep.Status.AvailableReplicas == *dep.Spec.Replicas {
		log.V(1).Info(
			"Updating Elastalert resources phase to SUCCESS.",
			"Elastalert.Namespace", o.elastalert.Namespace,
			"elastalert", o.elastalert.Name,
		)
		EmitK8sEvent(o.recorder, ea, corev1.EventTypeNormal, event.EventReasonSuccess, "Deployment has been stabilized.")
		return UpdateElastalertStatus(o.client, context.Background(), ea, esv1alpha1.ActionSuccess)

	}

	return nil
}

type Manager struct {
	observerLock sync.RWMutex
	observers    map[types.NamespacedName]*Observer
}

func NewManager() *Manager {
	return &Manager{
		observers: make(map[types.NamespacedName]*Observer),
	}
}

func (m *Manager) getObserver(key types.NamespacedName) (*Observer, bool) {
	m.observerLock.RLock()
	defer m.observerLock.RUnlock()

	observer, ok := m.observers[key]
	return observer, ok
}

func (m *Manager) Observe(elastalert *esv1alpha1.Elastalert, c client.Client, recorder record.EventRecorder) *Observer {
	nsName := types.NamespacedName{
		Namespace: elastalert.Namespace,
		Name:      elastalert.Name,
	}

	observer, exists := m.getObserver(nsName)
	if !exists {
		return m.createOrReplaceObserver(nsName, c, recorder)
	}
	return observer
}

// createOrReplaceObserver creates a new observer and adds it to the observers map, replacing existing observers if necessary.
func (m *Manager) createOrReplaceObserver(elastalert types.NamespacedName, c client.Client, recorder record.EventRecorder) *Observer {
	m.observerLock.Lock()
	defer m.observerLock.Unlock()

	observer := NewObserver(c, elastalert, esv1alpha1.ElastAlertObserveInterval, recorder)
	observer.Start()

	m.observers[elastalert] = observer
	return observer
}

func (m *Manager) StopObserving(key types.NamespacedName) {
	m.observerLock.Lock()
	defer m.observerLock.Unlock()

	if observer, ok := m.observers[key]; ok {
		observer.Stop()
		delete(m.observers, key)
	}
}

func UpdateElastalertStatus(c client.Client, ctx context.Context, e *esv1alpha1.Elastalert, flag string) error {
	condition := NewCondition(e, flag)
	if err := UpdateStatus(c, ctx, e, condition); err != nil {
		return err
	}
	return nil
}

func UpdateStatus(c client.Client, ctx context.Context, e *esv1alpha1.Elastalert, condition *metav1.Condition) error {
	patch := client.MergeFrom(e.DeepCopy())
	e.Status.Version = esv1alpha1.ElastAlertVersion

	if condition != nil {
		switch condition.Type {
		case esv1alpha1.ElastAlertAvailableType:
			e.Status.Phase = esv1alpha1.ElastAlertPhraseSucceeded
			meta.SetStatusCondition(&e.Status.Condictions, *condition)
			meta.RemoveStatusCondition(&e.Status.Condictions, esv1alpha1.ElastAlertUnAvailableType)
		case esv1alpha1.ElastAlertUnAvailableType:
			e.Status.Phase = esv1alpha1.ElastAlertPhraseFailed
			meta.SetStatusCondition(&e.Status.Condictions, *condition)
			meta.RemoveStatusCondition(&e.Status.Condictions, esv1alpha1.ElastAlertAvailableType)
		}
		if err := c.Status().Patch(ctx, e, patch); err != nil {
			log.Error(err, "Failed to update elastalert failed status", "Elastalert.Name", e.Name, "Status", e.Status.Phase)
			return err
		}
	}
	if len(e.Status.Condictions) == 0 && condition == nil || e.Status.Condictions[0].ObservedGeneration != e.Generation && condition == nil {
		e.Status.Phase = esv1alpha1.ElastAlertInitializing
		if err := c.Status().Patch(ctx, e, patch); err != nil {
			log.Error(err, "Failed to update elastalert failed status", "Elastalert.Name", e.Name, "Status", e.Status.Phase)
			return err
		}
	}
	log.V(1).Info(
		"Update Elastalert resources status success.",
		"Elastalert.Namespace", e.Name,
		"Status", e.Status.Phase,
	)
	return nil
}

func NewCondition(e *esv1alpha1.Elastalert, flag string) *metav1.Condition {
	var condition *metav1.Condition
	switch flag {
	case esv1alpha1.ActionSuccess:
		condition = &metav1.Condition{
			Type:               esv1alpha1.ElastAlertAvailableType,
			Status:             esv1alpha1.ElastAlertAvailableStatus,
			ObservedGeneration: e.Generation,
			LastTransitionTime: metav1.NewTime(podspec.GetUtcTime()),
			Reason:             esv1alpha1.ElastAlertAvailableReason,
			Message:            fmt.Sprintf("ElastAlert %s has successfully progressed.", e.Name),
		}
	case esv1alpha1.ActionFailed:
		condition = &metav1.Condition{
			Type:               esv1alpha1.ElastAlertUnAvailableType,
			Status:             esv1alpha1.ElastAlertUnAvailableStatus,
			ObservedGeneration: e.Generation,
			LastTransitionTime: metav1.NewTime(podspec.GetUtcTime()),
			Reason:             esv1alpha1.ElastAlertUnAvailableReason,
			Message:            fmt.Sprintf("Failed to apply ElastAlert %s resources.", e.Name),
		}
	case esv1alpha1.ResourcesCreating:
		return nil
	}
	return condition
}

func EmitK8sEvent(recorder record.EventRecorder, object runtime.Object, eventtype, reason, messageFmt string) {
	recorder.Eventf(object, eventtype, reason, messageFmt)
}
