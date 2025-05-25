package discoverytesting

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery/fake"
	clienttesting "k8s.io/client-go/testing"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
)

// FakeCachedDiscoveryClientBuilder is a builder for a discovery client.
type FakeCachedDiscoveryClientBuilder struct {
	// Groups contains the API groups to be returned by the discovery client.
	Groups []*metav1.APIGroup
	// Resources contains the API resources to be returned by the discovery client.
	Resources []*metav1.APIResourceList
	// PreferredResources contains the preferred API resources to be returned by the discovery client.
	PreferredResources []*metav1.APIResourceList
}

// NewFakeCachedDiscoveryClientBuilder creates a new FakeCachedDiscoveryClientBuilder.
func NewFakeCachedDiscoveryClientBuilder() *FakeCachedDiscoveryClientBuilder {
	return &FakeCachedDiscoveryClientBuilder{
		Resources:          []*metav1.APIResourceList{},
		Groups:             []*metav1.APIGroup{},
		PreferredResources: []*metav1.APIResourceList{},
	}
}

// CachedDiscoveryInterface returns the discovery client built by this builder.
func (c *FakeCachedDiscoveryClientBuilder) CachedDiscoveryInterface() *cmdtesting.FakeCachedDiscoveryClient {
	cached := cmdtesting.NewFakeCachedDiscoveryClient()
	cached.Groups = c.Groups
	cached.Resources = c.Resources
	cached.PreferredResources = c.PreferredResources
	cached.DiscoveryInterface = &fake.FakeDiscovery{
		Fake: &clienttesting.Fake{
			ReactionChain:      []clienttesting.Reactor{},
			WatchReactionChain: []clienttesting.WatchReactor{},
			ProxyReactionChain: []clienttesting.ProxyReactor{},

			Resources: c.Resources,
		},
		FakedServerVersion: &version.Info{},
	}

	return cached
}
