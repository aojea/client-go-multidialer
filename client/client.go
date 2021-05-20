package client

import (
	"context"

	"github.com/aojea/client-go-multidialer/multidialer"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// NewForConfig creates a resilient client-go that, in case of connection failures,
// tries to connect to all the available apiservers in the cluster.
func NewForConfig(ctx context.Context, config *rest.Config) (*kubernetes.Clientset, error) {
	// create the clientset
	configShallowCopy := *config
	// it wraps the custom dialer if exists
	d := multidialer.NewDialer(configShallowCopy.Dial)
	// use the multidialier for our clientset
	configShallowCopy.Dial = d.DialContext
	// create the clientset with our own dialer
	cs, err := kubernetes.NewForConfig(&configShallowCopy)
	if err != nil {
		return cs, err
	}
	// start the resolver to update the list of available apiservers
	// !!! using our own dialer !!!
	d.Start(ctx, cs)
	return cs, nil
}
