package trackers

import (
	"context"
	"fmt"
	coreapiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
	"k8s.io/klog"
	"secrets-manager/pkg/kube"
)

var revisionAnnotation = "base-revision"
var managedNamespaceLabel = "ops.dev/secret-sync"

type secretCh struct {
	namespace string
	object    *coreapiv1.Secret
}

type SecretFeed struct {
	namespace string
	onAdd     chan *secretCh
	onUpdate  chan *secretCh
}

func NewSecretFeed(namespace string) *SecretFeed {
	return &SecretFeed{
		namespace: namespace,
		onAdd:     make(chan *secretCh),
		onUpdate:  make(chan *secretCh),
	}
}

func (s *SecretFeed) Run(ctx context.Context) {
	s.runInformer(ctx)
	s.handleEvent()
}

func (s *SecretFeed) handleEvent() {
	go func() {
		for {
			select {
			case update := <-s.onUpdate:
				_, err := kube.Kubernetes.CoreV1().Secrets("default").Update(copySecret(update.object, update.namespace))
				if err != nil {
					klog.Infof("update secret name=%s namespace=%s fail: %s", update.object.Name, update.namespace, err)
				}
				klog.Infof("update secret name=%s namespace=%s success", update.object.Name, update.namespace)
			case add := <-s.onAdd:
				_, err := kube.Kubernetes.CoreV1().Secrets("default").Create(copySecret(add.object, add.namespace))
				if err != nil {
					klog.Infof("create secret name=%s namespace=%s fail: %s", add.object.Name, add.namespace, err)
				}
				klog.Infof("create secret name=%s namespace=%s success", add.object.Name, add.namespace)
			}
		}
	}()
}

func (s *SecretFeed) runInformer(ctx context.Context) {
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return kube.Kubernetes.CoreV1().Secrets(s.namespace).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return kube.Kubernetes.CoreV1().Secrets(s.namespace).Watch(options)
		},
	}
	go func() {
		watchtools.UntilWithSync(ctx, lw, &coreapiv1.Secret{}, nil,
			func(event watch.Event) (bool, error) {
				var object *coreapiv1.Secret
				var ok bool
				object, ok = event.Object.(*coreapiv1.Secret)
				if !ok {
					return false, fmt.Errorf("TRACK EVENT expect *corev1.Secret object, got %T", event.Object)
				}
				if exclude(object) {
					klog.Infof("ignore event for %s", object.Type)
					return false, nil
				}
				klog.Infof("watch event type=%s name=%s kind=%s namespace=%s", event.Type, object.Name, object.Type, object.Namespace)
				namespaces, err := managedNamespaces()
				if err != nil {
					klog.Errorf("List managed namespaces with label %s fail", managedNamespaceLabel)
				}
				switch event.Type {
				case watch.Added, watch.Modified:
					if namespaces.Items == nil {
						return false, nil
					}
					for _, ns := range namespaces.Items {
						target, err := kube.Kubernetes.CoreV1().Secrets(ns.Name).Get(object.Name, metav1.GetOptions{})
						var create bool
						if err != nil {
							klog.Errorf("get target secret name=%s namespace=%s err=%s", object.Name, ns.Name, err)
							if statusError, isStatus := err.(*errors.StatusError); isStatus {
								create = statusError.ErrStatus.Reason == metav1.StatusReasonNotFound
							}
						}
						if create {
							s.onAdd <- &secretCh{
								namespace: ns.Name,
								object:    object,
							}
						} else if !upToDate(object, target) {
							s.onUpdate <- &secretCh{
								namespace: ns.Name,
								object:    object,
							}
						}
					}

				}
				return false, nil
			})
	}()
}

func upToDate(source, dest *coreapiv1.Secret) bool {
	if source == nil || dest == nil {
		return false
	}
	return source.ResourceVersion == dest.Annotations["baseResourceVersion"]
}

func copySecret(secret *coreapiv1.Secret, namespace string) *coreapiv1.Secret {
	annotations := secret.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[revisionAnnotation] = secret.ResourceVersion
	return &coreapiv1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:        secret.Name,
			Namespace:   namespace,
			Labels:      secret.ObjectMeta.Labels,
			Annotations: annotations,
		},
		Data: secret.Data,
		Type: secret.Type,
	}
}

func managedNamespaces() (*coreapiv1.NamespaceList, error) {
	selector := metav1.LabelSelector{
		MatchLabels: map[string]string{managedNamespaceLabel: "enabled"},
	}
	labelMap, _ := metav1.LabelSelectorAsMap(&selector)
	return kube.Kubernetes.CoreV1().Namespaces().List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	})
}

func exclude(secret *coreapiv1.Secret) bool {
	return secret.Type != coreapiv1.SecretTypeDockerConfigJson && secret.Type != coreapiv1.SecretTypeTLS &&
		secret.Type != coreapiv1.SecretTypeBootstrapToken && secret.Type != coreapiv1.SecretTypeOpaque
}
