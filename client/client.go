package client

import (
	"context"

	"github.com/aojea/client-go-multidialer/multidialer"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// NewClient creates a resilient client-go that, in case of connection failures,
// tries to connect to all the available apiservers in the cluster.
func NewClient(ctx context.Context, config *rest.Config) (*kubernetes.Clientset, error) {
	// create the clientset
	configShallowCopy := *config
	d := multidialer.NewDialer()
	// wrap the custom dialer if exist
	if configShallowCopy.Dial != nil {
		d.DialFunc = configShallowCopy.Dial
	}
	// use the multidialier for our clientset
	configShallowCopy.Dial = d.DialContext
	cs, err := kubernetes.NewForConfig(&configShallowCopy)
	if err != nil {
		return cs, err
	}
	// start the resolver to update the list of available apiservers
	d.Start(ctx, cs)
	return cs, nil
}
