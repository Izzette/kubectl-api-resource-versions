package cmd

import (
	"bytes"

	"github.com/Izzette/kubectl-api-resource-versions/internal/discoverytesting"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/discovery"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
)

// APIResourceVersionsOptionsBuilder is a builder for [apiResourceVersionsOptions].
type APIResourceVersionsOptionsBuilder struct {
	options               *apiResourceVersionsOptions
	discoveryClient       *cmdtesting.FakeCachedDiscoveryClient
	stdin, stdout, stderr *bytes.Buffer
}

// NewTestOptionsBuilder creates a new APIResourceVersionsOptionsBuilder for testing purposes.
// The discovery client is initialized with a small testing dataset, see [discoverytesting.New] for more details.
func NewTestOptionsBuilder() *APIResourceVersionsOptionsBuilder {
	ioStreams, stdin, stdout, stderr := genericiooptions.NewTestIOStreams()
	options := newAPIResourceVersionsOptions(ioStreams)
	discoveryClient := discoverytesting.New()
	options.discoveryClient = discoveryClient
	builder := &APIResourceVersionsOptionsBuilder{
		options:         options,
		discoveryClient: discoveryClient,
		stdin:           stdin,
		stdout:          stdout,
		stderr:          stderr,
	}

	return builder
}

// APIResourceVersionsOptions returns the [apiResourceVersionsOptions] built by this builder.
func (o *APIResourceVersionsOptionsBuilder) APIResourceVersionsOptions() *apiResourceVersionsOptions {
	return o.options
}

// GetBuffers returns the input, output, and error buffers for the io streams provided in the options.
func (o *APIResourceVersionsOptionsBuilder) GetBuffers() (*bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	return o.stdin, o.stdout, o.stderr
}

// WithDiscoveryClient overrides the discovery client for the options.
func (o *APIResourceVersionsOptionsBuilder) WithDiscoveryClient(
	discoveryClient discovery.CachedDiscoveryInterface,
) *APIResourceVersionsOptionsBuilder {
	o.discoveryClient = nil
	o.options.discoveryClient = discoveryClient

	return o
}

// SetOutput sets the output format for the options, see [apiResourceVersionsOptions.Output].
func (o *APIResourceVersionsOptionsBuilder) SetOutput(output string) *APIResourceVersionsOptionsBuilder {
	o.options.Output = output

	return o
}

// SetSortBy sets the sort order for the options, see [apiResourceVersionsOptions.SortBy].
func (o *APIResourceVersionsOptionsBuilder) SetSortBy(sortBy string) *APIResourceVersionsOptionsBuilder {
	o.options.SortBy = sortBy

	return o
}

// SetAPIGroup sets the API group for the options, see [apiResourceVersionsOptions.APIGroup].
func (o *APIResourceVersionsOptionsBuilder) SetAPIGroup(apiGroup string) *APIResourceVersionsOptionsBuilder {
	o.options.APIGroup = apiGroup
	o.options.groupChanged = true

	return o
}

// SetNamespaced sets whether the resources are namespaced or not, see [apiResourceVersionsOptions.Namespaced].
func (o *APIResourceVersionsOptionsBuilder) SetNamespaced(namespaced bool) *APIResourceVersionsOptionsBuilder {
	o.options.Namespaced = namespaced
	o.options.nsChanged = true

	return o
}

// SetVerbs sets the verbs for the options, see [apiResourceVersionsOptions.Verbs].
func (o *APIResourceVersionsOptionsBuilder) SetVerbs(verbs []string) *APIResourceVersionsOptionsBuilder {
	o.options.Verbs = verbs

	return o
}

// SetNoHeaders sets whether to print headers or not, see [apiResourceVersionsOptions.NoHeaders].
func (o *APIResourceVersionsOptionsBuilder) SetNoHeaders(noHeaders bool) *APIResourceVersionsOptionsBuilder {
	o.options.NoHeaders = noHeaders

	return o
}

// SetCached sets whether to use a cached discovery client or not, see [apiResourceVersionsOptions.Cached].
func (o *APIResourceVersionsOptionsBuilder) SetCached(cached bool) *APIResourceVersionsOptionsBuilder {
	o.options.Cached = cached

	return o
}

// SetCategories sets the categories for the options, see [apiResourceVersionsOptions.Categories].
func (o *APIResourceVersionsOptionsBuilder) SetCategories(categories []string) *APIResourceVersionsOptionsBuilder {
	o.options.Categories = categories

	return o
}

// SetPreferred sets whether to prefer the preferred version of the resources, see
// [apiResourceVersionsOptions.Preferred].
func (o *APIResourceVersionsOptionsBuilder) SetPreferred(preferred bool) *APIResourceVersionsOptionsBuilder {
	o.options.Preferred = preferred
	o.options.preferredChanged = true

	return o
}
