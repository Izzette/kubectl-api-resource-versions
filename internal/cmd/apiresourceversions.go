/*
Copyright 2025 Isabelle COWAN-BERGMAN
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/liggitt/tabwriter"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apimachineryerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/discovery"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Enable all auth plugins (for CSPs)
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/templates"
)

const (
	wideOutput = "wide"
	nameOutput = "name"

	nameSortBy = "name"
	kindSortBy = "kind"
)

var (
	// apiresourceversionsExample is the example text for the api-resource-versions command.
	//
	//nolint:gochecknoglobals
	apiresourceversionsExample = `
		# Print all API resources with their group versions (including deprecated or unstable versions)
		kubectl api-resource-versions

		# Print in the 'name' format for use with kubectl get
		kubectl api-resource-versions --output=name

		# Filter to non-preferred versions
		kubectl api-resource-versions --preferred=false

		# Filter to resources in the apps group
		kubectl api-resource-versions --api-group=apps

		# List all non-namespaced resources
		kubectl api-resource-versions --namespaced=false`
)

// NewCmdAPIResourceVersions returns a command that lists all API resources and their versions.
//
// TODO(Izzette): Output only supports default, wide, and name; it would be interesting to export to JSON or YAML.
// TODO(Izzette): Subresources are not included in the output; they are potentially useful, but it's unclear how to
// expose them in a useful, machine-readable output.
func NewCmdAPIResourceVersions(
	configFlags *genericclioptions.ConfigFlags,
	ioStreams genericiooptions.IOStreams,
) *cobra.Command {
	options := newAPIResourceVersionsOptions(ioStreams)

	cmd := &cobra.Command{
		Use:   "api-resource-versions",
		Short: "List all API resources and versions",
		Long: "List all API resources and their API group versions along with whether the version is preferred.\n" +
			"Subresources are not included.",
		Example: templates.Examples(apiresourceversionsExample),
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(options.complete(configFlags, cmd, args))
			cmdutil.CheckErr(options.validate())
			cmdutil.CheckErr(runAPIResourceVersions(options))
		},
	}

	cmd.Flags().BoolVar(&options.NoHeaders, "no-headers", options.NoHeaders,
		"When using the default or custom-column output format, don't print headers (default print headers).")
	cmd.Flags().StringVarP(&options.Output, "output", "o", options.Output,
		"Output format. One of: ("+wideOutput+", "+nameOutput+").")

	cmd.Flags().StringVar(&options.APIGroup, "api-group", options.APIGroup,
		"Limit to resources in the specified API group.")
	cmd.Flags().BoolVar(&options.Namespaced, "namespaced", options.Namespaced,
		"If false, non-namespaced resources will be returned, otherwise returning namespaced resources by default.")
	cmd.Flags().StringSliceVar(&options.Verbs, "verbs", options.Verbs,
		"Limit to resources that support the specified verbs.")
	cmd.Flags().StringVar(&options.SortBy, "sort-by", options.SortBy,
		"If non-empty, sort list of resources using specified field. One of ("+nameSortBy+", "+kindSortBy+").")
	cmd.Flags().BoolVar(&options.Cached, "cached", options.Cached, "Use the cached list of resources if available.")
	cmd.Flags().StringSliceVar(&options.Categories, "categories", options.Categories,
		"Limit to resources that belong to the specified categories.")
	cmd.Flags().BoolVar(&options.Preferred, "preferred", options.Preferred,
		"Filter resources by whether their version is in the server preferred resources.")
	configFlags.AddFlags(cmd.Flags())

	return cmd
}

// apiResourceVersionsOptions contains the options for the api-resource-versions command.
type apiResourceVersionsOptions struct {
	genericiooptions.IOStreams

	Output     string
	SortBy     string
	APIGroup   string
	Namespaced bool
	Verbs      []string
	NoHeaders  bool
	Cached     bool
	Categories []string
	Preferred  bool

	groupChanged     bool
	nsChanged        bool
	preferredChanged bool

	discoveryClient discovery.CachedDiscoveryInterface
}

// newAPIResourceVersionsOptions returns a new [apiResourceVersionsOptions] with default values.
func newAPIResourceVersionsOptions(ioStreams genericiooptions.IOStreams) *apiResourceVersionsOptions {
	return &apiResourceVersionsOptions{
		IOStreams:  ioStreams,
		Namespaced: true,
	}
}

// groupResource is represents a versioned API resource.
type groupResource struct {
	// APIGroup is the API group of the resource.
	APIGroup *metav1.APIGroup
	// APIGroupVersion is the group version of the resource, e.g. "apps/v1".
	APIGroupVersion string
	// APIResource is the API resource for this version.
	APIResource *metav1.APIResource
	// Preferred if this is the preferred version for the resource.
	Preferred bool
}

// PreferredGroupVersion returns true if the version is the preferred version for the API group.
// Note that this is not the same as a preferred version for the resource.
func (gr groupResource) PreferredGroupVersion() bool {
	return gr.APIGroup.PreferredVersion.GroupVersion == gr.APIGroupVersion
}

// fullname returns the name of the resource with its version and api group in the format expected by kubectl.
func (gr groupResource) fullname() string {
	groupVersion, err := schema.ParseGroupVersion(gr.APIGroupVersion)
	if err != nil {
		panic(fmt.Errorf("error parsing group version: %w", err))
	} else if groupVersion.Group != gr.APIGroup.Name {
		panic(fmt.Sprintf("group version %s does not match group %s", groupVersion.Group, gr.APIGroup.Name))
	}

	return fmt.Sprintf("%s.%s.%s", gr.APIResource.Name, groupVersion.Version, gr.APIGroup.Name)
}

// errWrongOutput is a returned when the output format is not supported.
const errWrongOutput = constError("output must be one of: (" + wideOutput + ", " + nameOutput + ")")

// errSortBy is a returned when the sort-by field is not supported.
const errSortBy = constError("sort-by must be one of: (" + nameSortBy + ", " + kindSortBy + ")")

// validate checks that options are valid for the command.
func (o *apiResourceVersionsOptions) validate() error {
	supportedOutputTypes := sets.New("", wideOutput, nameOutput)
	if !supportedOutputTypes.Has(o.Output) {
		return fmt.Errorf("%w: %s is not available", errWrongOutput, o.Output)
	}

	supportedSortTypes := sets.New("", nameSortBy, kindSortBy)
	if len(o.SortBy) > 0 {
		if !supportedSortTypes.Has(o.SortBy) {
			return fmt.Errorf("%w: %s is not available", errSortBy, o.SortBy)
		}
	}

	return nil
}

// complete completes all the required options for the api-resource-versions command.
func (o *apiResourceVersionsOptions) complete(
	restClientGetter genericclioptions.RESTClientGetter,
	cmd *cobra.Command,
	args []string,
) error {
	if len(args) != 0 {
		//nolint:wrapcheck
		return cmdutil.UsageErrorf(cmd, "unexpected arguments: %v", args)
	}

	discoveryClient, err := restClientGetter.ToDiscoveryClient()
	if err != nil {
		return fmt.Errorf("couldn't create discovery client: %w", err)
	}

	o.discoveryClient = discoveryClient

	o.groupChanged = cmd.Flags().Changed("api-group")
	o.nsChanged = cmd.Flags().Changed("namespaced")
	o.preferredChanged = cmd.Flags().Changed("preferred")

	return nil
}

// errNoResourcesFound is a constant error returned when no resources are found.
const errNoResourcesFound = constError("no resources found")

// runAPIResourceVersions prints the API resources and their group versions.
func runAPIResourceVersions(options *apiResourceVersionsOptions) error {
	resources, err := getGroupResources(options)
	if err != nil {
		return err
	}

	if len(resources) == 0 && options.Output != nameOutput {
		// If no resources are found, we return an error.
		return errNoResourcesFound
	}

	return printGroupResources(resources, options)
}

// splitResourceName splits the resource name into its resource and subresource parts.
// The first return value is the resource name, and the second return value is the subresource name if it exists.
func splitResourceName(resourceName string) (string, *string) {
	//nolint:mnd
	parts := strings.SplitN(resourceName, "/", 2)
	if len(parts) == 1 {
		// If there is no subresource, we return the resource name and an empty string.
		return parts[0], nil
	}

	subresourceName := parts[1]

	return parts[0], &subresourceName
}

// unversionedResourceName returns the resource name in the format expected by kubectl, without the version.
// The first return value is the resource name with its group in the format "<resource>.<group>".
// The second return value is the subresource name if it exists.
func unversionedResourceName(resource metav1.APIResource) (string, *string) {
	// If the resource is a subresource, we skip it.
	resourceName, subresourceName := splitResourceName(resource.Name)

	// Otherwise, we return the resource name with its group.
	return fmt.Sprintf("%s.%s", resourceName, resource.Group), subresourceName
}

// getPreferredResourceVersions retrieves the server preferred resource versions.
// For the returned map:
//
//   - The key is in the format returned by [unversionedResourceName], which is "<resource>.<group>".
//   - The value is the version for the group (e.g. "v1", "v1beta1", "v2").
//
// Subresources are not included in the map.
func getPreferredResourceVersions(options *apiResourceVersionsOptions) (map[string]string, error) {
	preferredResources, err := options.discoveryClient.ServerPreferredResources()
	if err != nil {
		return nil, fmt.Errorf("couldn't get server preferred resources: %w", err)
	}

	preferredVersions := make(map[string]string, len(preferredResources))
	for _, resourceList := range preferredResources {
		groupVersion, err := schema.ParseGroupVersion(resourceList.GroupVersion)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse group version %s: %w", resourceList.GroupVersion, err)
		}

		for _, resource := range resourceList.APIResources {
			resource.Group = groupVersion.Group // Why is this not set?

			resourceKey, subresourceName := unversionedResourceName(resource)
			if subresourceName != nil {
				// If the resource is a subresource, we skip it.
				continue
			}

			preferredVersions[resourceKey] = groupVersion.Version
		}
	}

	return preferredVersions, nil
}

// getGroupResources retrieves the API resources and their group versions from the discovery client.
func getGroupResources(options *apiResourceVersionsOptions) ([]groupResource, error) {
	if !options.Cached {
		options.discoveryClient.Invalidate()
	}

	groupList, err := options.discoveryClient.ServerGroups()
	if err != nil {
		return []groupResource{}, fmt.Errorf("couldn't get server groups: %w", err)
	}

	preferredResources, err := getPreferredResourceVersions(options)
	if err != nil {
		return nil, fmt.Errorf("couldn't get preferred resource versions: %w", err)
	}

	// We could quickly calculate the total number of resources in the server groups to avoid having to re-size the
	// underlying slice-buffer during an append operation.
	// However, when the number of resources is large, this could result in very high memory usage even when heavily
	// filtering the group resources.
	resources := make([]groupResource, 0)

	for i := range groupList.Groups {
		group := &groupList.Groups[i]

		if excludeGroup(group, options) {
			// If the group is excluded, we skip it.
			continue
		}

		groupResources, err := processGroupResources(options, group, preferredResources)
		if err != nil {
			return nil, err
		}

		resources = append(resources, groupResources...)
	}

	return resources, nil
}

func processGroupResources(
	options *apiResourceVersionsOptions,
	group *metav1.APIGroup,
	preferredResources map[string]string,
) ([]groupResource, error) {
	resources := make([]groupResource, 0)

	for _, version := range group.Versions {
		resourceList, err := options.discoveryClient.ServerResourcesForGroupVersion(version.GroupVersion)
		if err != nil {
			return nil, fmt.Errorf("couldn't get server resources for group version %s: %w", version.GroupVersion, err)
		}

		for _, apiResource := range resourceList.APIResources {
			apiResource.Group = group.Name // Why is this not set?

			resourceName, subresourceName := unversionedResourceName(apiResource)
			if subresourceName != nil {
				// If the resource is a subresource, we skip it.
				continue
			}

			preferredVersion, ok := preferredResources[resourceName]
			preferred := ok && preferredVersion == version.Version

			resource := groupResource{
				APIGroup:        group,
				APIGroupVersion: version.GroupVersion,
				APIResource:     &apiResource,
				Preferred:       preferred,
			}

			if !excludeGroupResource(resource, options) {
				resources = append(resources, resource)
			}
		}
	}

	return resources, nil
}

// excludeGroup checks if the group should be excluded based on the options.
func excludeGroup(group *metav1.APIGroup, options *apiResourceVersionsOptions) bool {
	if options.groupChanged && options.APIGroup != group.Name {
		return true
	}

	return false
}

// excludeGroupResource checks if the resource should be excluded based on the options.
//
//nolint:cyclop
func excludeGroupResource(resource groupResource, options *apiResourceVersionsOptions) bool {
	if strings.Contains(resource.APIResource.Name, "/") {
		// If the resource name contains a slash, it is a subresource and we skip it.
		return true
	}

	if options.nsChanged && options.Namespaced != resource.APIResource.Namespaced {
		return true
	}

	if len(options.Verbs) > 0 && !sets.New(resource.APIResource.Verbs...).HasAll(options.Verbs...) {
		return true
	}

	if len(options.Categories) > 0 && !sets.New(resource.APIResource.Categories...).HasAll(options.Categories...) {
		return true
	}

	if options.preferredChanged && options.Preferred != resource.Preferred {
		return true
	}

	return false
}

// printGroupResources prints the API resources and their group versions in the format specified by
// [apiResourceVersionsOptions].
func printGroupResources(resources []groupResource, options *apiResourceVersionsOptions) error {
	writer := printers.GetNewTabWriter(options.Out)
	defer mustFlushWriter(writer)

	if !options.NoHeaders && options.Output != nameOutput {
		err := printHeaders(writer, options.Output)
		if err != nil {
			return err
		}
	}

	sort.Stable(sortableResource{resources, options.SortBy})

	var errs []error

	for _, resource := range resources {
		var err error

		switch options.Output {
		case nameOutput:
			err = printGroupResourcesByName(writer, resource)
		case wideOutput:
			err = printGroupResourcesWide(writer, resource)
		default:
			err = printGroupResourcesDefault(writer, resource)
		}

		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return apimachineryerrors.NewAggregate(errs)
	}

	return nil
}

// printHeaders prints the headers for the output table.
func printHeaders(out io.Writer, output string) error {
	headers := []string{"NAME", "SHORTNAMES", "APIVERSION", "NAMESPACED", "KIND", "PREFERRED"}
	if output == "wide" {
		headers = append(headers, "GROUPPREFERRED", "VERBS", "CATEGORIES")
	}

	_, err := fmt.Fprintf(out, "%s\n", strings.Join(headers, "\t"))
	if err != nil {
		return fmt.Errorf("error printing headers: %w", err)
	}

	return nil
}

// printGroupResourcesByName prints the API resource name in the format expected by kubectl.
func printGroupResourcesByName(writer io.Writer, resource groupResource) error {
	_, err := fmt.Fprintf(writer, "%s\n", resource.fullname())
	if err != nil {
		return fmt.Errorf("error printing resource name: %w", err)
	}

	return nil
}

// printGroupResourcesWide prints the API resources in wide format.
func printGroupResourcesWide(writer io.Writer, resource groupResource) error {
	_, err := fmt.Fprintf(writer, "%s\t%s\t%s\t%v\t%s\t%t\t%t\t%s\t%s\n",
		resource.APIResource.Name,
		strings.Join(resource.APIResource.ShortNames, ","),
		resource.APIGroupVersion,
		resource.APIResource.Namespaced,
		resource.APIResource.Kind,
		resource.Preferred,
		resource.PreferredGroupVersion(),
		strings.Join(resource.APIResource.Verbs, ","),
		strings.Join(resource.APIResource.Categories, ","),
	)
	if err != nil {
		return fmt.Errorf("error printing resource in wide format: %w", err)
	}

	return nil
}

// printGroupResourcesDefault prints the API resources in the default format.
func printGroupResourcesDefault(writer io.Writer, resource groupResource) error {
	_, err := fmt.Fprintf(writer, "%s\t%s\t%s\t%v\t%s\t%t\n",
		resource.APIResource.Name,
		strings.Join(resource.APIResource.ShortNames, ","),
		resource.APIGroupVersion,
		resource.APIResource.Namespaced,
		resource.APIResource.Kind,
		resource.Preferred,
	)
	if err != nil {
		return fmt.Errorf("error printing resource in default format: %w", err)
	}

	return nil
}

// mustFlushWriter flushes the writer to ensure all data is written.
func mustFlushWriter(writer *tabwriter.Writer) {
	err := writer.Flush()
	if err != nil {
		panic(fmt.Errorf("error flushing writer: %w", err))
	}
}

// sortableResource implements [sort.Interface] for [[]groupResource] based on the specified field.
type sortableResource struct {
	resources []groupResource
	sortBy    string
}

// Len implements [sort.Interface.Len] for [sortableResource].
func (s sortableResource) Len() int {
	return len(s.resources)
}

// Swap implements [sort.Interface.Swap] for [sortableResource].
func (s sortableResource) Swap(i, j int) {
	s.resources[i], s.resources[j] = s.resources[j], s.resources[i]
}

// Less implements [sort.Interface.Less] for [sortableResource].
func (s sortableResource) Less(i, j int) bool {
	left, right := s.resources[i], s.resources[j]

	switch s.sortBy {
	case nameSortBy:
		return left.APIResource.Name < right.APIResource.Name
	case kindSortBy:
		return left.APIResource.Kind < right.APIResource.Kind
	default:
		if left.APIGroup.Name != right.APIGroup.Name {
			return left.APIGroup.Name < right.APIGroup.Name
		}

		return left.APIResource.Name < right.APIResource.Name
	}
}

// constError is a simple implementation of the error interface that returns a constant string.
type constError string

// Error implements the error interface.
func (e constError) Error() string {
	return string(e)
}
