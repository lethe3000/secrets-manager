package trackers

import (
	"context"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	signals "secrets-manager/pkg/signal"
)

type Tracker struct {
	Secret *SecretFeed

	client           kubernetes.Interface
	watchedNamespace string
}

func NewTracker(client kubernetes.Interface, watched string) *Tracker {
	return &Tracker{
		client:           client,
		watchedNamespace: watched,
	}
}

func (t *Tracker) InitResource() {
	t.Secret = NewSecretFeed(t.watchedNamespace)
}

func (t *Tracker) Run() {
	ctx, _ := context.WithCancel(context.Background())
	t.Secret.Run(ctx)
	// wait to signal exit
	stopCh := signals.SetupSignalHandler()
	select {
	case <-stopCh:
		klog.Info("exit")
	}
}
