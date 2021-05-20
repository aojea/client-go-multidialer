package multidialer

import (
	"context"
	"fmt"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// resolver updates a local cache with current apiserver endpoints
// it stores the apiserver endpoints in the ip:port format
// and tracks which was the last apiserver connected successfully
type resolver struct {
	mu sync.Mutex
	// host url and state
	// TODO: we can set the status of the host in the cache if needed
	// i.e. apiserver is up but not ready equal to false, so we don't
	// try to connect against it.
	cache map[string]bool
	// last apiserver connected successfully
	last string
}

// NewResolver returns a hosts pool to control the dialer destination
func NewResolver(alternateHosts []string) *resolver {
	hosts := map[string]bool{}
	if len(alternateHosts) > 0 {
		for _, h := range alternateHosts {
			hosts[h] = true
		}
	}
	return &resolver{
		cache: hosts,
	}
}

// setLast records the last host successfully connected
func (r *resolver) setLast(host string) {
	r.mu.Lock()
	r.last = host
	r.mu.Unlock()
}

// updateCache updates the cache with a new list of apiserver endpoints
func (r *resolver) updateCache(hosts map[string]bool) {
	r.mu.Lock()
	r.cache = map[string]bool{}
	for h, _ := range hosts {
		r.cache[h] = true
	}
	r.mu.Unlock()
}

// listReady returns an ordered list only with the hosts that are ready
// the first host in the list is the last one we successfully connected
func (r *resolver) listReady() []string {
	hosts := []string{}
	r.mu.Lock()
	for k, v := range r.cache {
		// skip disabled hosts
		if !v {
			continue
		}
		// prepend if is the last good one
		// so we try to connect against it first
		if k == r.last {
			hosts = append([]string{k}, hosts...)
		} else {
			hosts = append(hosts, k)
		}
	}
	r.mu.Unlock()
	return hosts
}

// start starts a loop to get the apiserver endpoints from the apiserver
// so the dialer can connect to the registered apiservers in the cluster.
// This is the tricky part, since the resolver uses the same dialer it feeds
// so it will benefit from the resilience it provides.
func (r *resolver) start(ctx context.Context, clientset kubernetes.Interface) {
	// run a goroutine updating the apiserver hosts in the dialer
	// this handle cluster resizing and renumbering
	go func() {
		// add the list of alternate hosts to the dialer obtained from the apiserver endpoints
		// TODO: use a custom interval
		// TODO: use watchers? I feel this is more resilient
		tick := time.Tick(60 * time.Second)
		for {
			select {
			case <-tick:
				// apiservers are registered as endpoints of the kubernetes.default service
				endpoint, err := clientset.CoreV1().Endpoints("default").Get(context.TODO(), "kubernetes", metav1.GetOptions{})
				if err != nil || len(endpoint.Subsets) == 0 {
					continue
				}
				newHosts := map[string]bool{}
				// get current hosts
				for _, ss := range endpoint.Subsets {
					for _, e := range ss.Addresses {
						host := fmt.Sprintf("%s:%d", e.IP, ss.Ports[0].Port)
						newHosts[host] = true
					}
				}
				// update the cache with the new hosts
				r.updateCache(newHosts)
			case <-ctx.Done():
				return
			}

		}
	}()
}
