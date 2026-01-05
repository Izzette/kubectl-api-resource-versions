package discoverytesting

import (
	_ "embed"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
)

// New returns a new [cmdtesting.FakeCachedDiscoveryClient] with a small set of API groups and resources.
// It includes some of the core group and the autoscaling group, with their respective resources.
func New() *cmdtesting.FakeCachedDiscoveryClient {
	cached := NewFakeCachedDiscoveryClientBuilder()

	cached.Groups = append(cached.Groups, getCoreGroup(), getAutoscalingGroup())

	cached.Resources = append(cached.Resources, getCoreResources()...)
	cached.Resources = append(cached.Resources, getAutoscalingResources()...)

	cached.PreferredResources = append(cached.PreferredResources, getCorePreferredResources())
	cached.PreferredResources = append(cached.PreferredResources, getAutoscalingPreferredResources())

	return cached.CachedDiscoveryInterface()
}

// NewProcedural creates a new [cmdtesting.FakeCachedDiscoveryClient] with a procedural generation of API groups,
// versions, and resources.
// The groups will be named "group0", "group1", etc., and each group will have a specified number of versions.
// Each group will have versions named "v1", "v2", etc., with the highest version number of being set as the preferred
// version.
// Each group version will have a specified number of resources, named "resource0", "resource1", etc..
// Each resource will be namespaced, have the verbs "get", "list", and "watch", and belong to the "all" category.
// No subresources will be included.
// Only the resources from the preferred version of each group will be included in the preferred resources list.
func NewProcedural(groups, versionsPerGroup, resourcesPerVersion int) *cmdtesting.FakeCachedDiscoveryClient {
	builder := NewFakeCachedDiscoveryClientBuilder()

	for i := range groups {
		groupName := fmt.Sprintf("group%d", i)
		group := &metav1.APIGroup{
			Name:     groupName,
			Versions: []metav1.GroupVersionForDiscovery{},
		}

		for j := versionsPerGroup; j > 0; j-- {
			versionName := fmt.Sprintf("v%d", j)
			group.Versions = append(group.Versions, metav1.GroupVersionForDiscovery{
				GroupVersion: groupName + "/" + versionName,
				Version:      versionName,
			})
		}

		group.PreferredVersion = group.Versions[0] // Set the first version as preferred
		builder.Groups = append(builder.Groups, group)
	}

	for _, group := range builder.Groups {
		resources := make([]metav1.APIResource, 0, resourcesPerVersion)
		for i := range resourcesPerVersion {
			resourceName := fmt.Sprintf("resource%d", i)
			resource := metav1.APIResource{
				Name:       resourceName,
				Namespaced: true,
				Verbs:      []string{"get", "list", "watch"},
				Categories: []string{"all"},
			}
			resources = append(resources, resource)
		}

		for _, version := range group.Versions {
			apiResourceList := &metav1.APIResourceList{
				GroupVersion: version.GroupVersion,
				APIResources: []metav1.APIResource{},
			}
			apiResourceList.APIResources = append(apiResourceList.APIResources, resources...)
			builder.Resources = append(builder.Resources, apiResourceList)
		}
	}

	builder.Resources = append(builder.Resources, getCoreResources()...)
	builder.Resources = append(builder.Resources, getAutoscalingResources()...)

	builder.PreferredResources = append(builder.PreferredResources, getCorePreferredResources())
	builder.PreferredResources = append(builder.PreferredResources, getAutoscalingPreferredResources())

	return builder.CachedDiscoveryInterface()
}
