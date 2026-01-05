package discoverytesting

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/Izzette/kubectl-api-resource-versions/internal/yamlutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8syaml "sigs.k8s.io/yaml"
)

//go:embed testdata/core-group.yaml
var coreGroupYAML []byte

//go:embed testdata/core-resources.yaml
var coreResourcesYAML []byte

//go:embed testdata/autoscaling-group.yaml
var autoscalingGroupYAML []byte

//go:embed testdata/autoscaling-resources.yaml
var autoscalingResourcesYAML []byte

func getCoreGroup() *metav1.APIGroup {
	return getGroup(coreGroupYAML)
}

func getCoreResources() []*metav1.APIResourceList {
	return getResources(coreResourcesYAML)
}

func getCorePreferredResources() *metav1.APIResourceList {
	return getPreferredResources(getCoreGroup(), getCoreResources())
}

func getAutoscalingGroup() *metav1.APIGroup {
	return getGroup(autoscalingGroupYAML)
}

func getAutoscalingResources() []*metav1.APIResourceList {
	return getResources(autoscalingResourcesYAML)
}

func getAutoscalingPreferredResources() *metav1.APIResourceList {
	return getPreferredResources(getAutoscalingGroup(), getAutoscalingResources())
}

func getGroup(groupYAML []byte) *metav1.APIGroup {
	groupJSON, err := k8syaml.YAMLToJSON(groupYAML)
	if err != nil {
		panic(fmt.Errorf("failed to convert group YAML to JSON: %w", err))
	}

	group := &metav1.APIGroup{}

	err = json.Unmarshal(groupJSON, &group)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal group: %w", err))
	}

	return group
}

func getResources(resourcesYAML []byte) []*metav1.APIResourceList {
	resourcesYAMLBuf := bytes.NewBuffer(resourcesYAML)

	resources := make([]*metav1.APIResourceList, 0)

	for result := range yamlutil.YAMLDocumentsToJSON(resourcesYAMLBuf) {
		decoder, err := result.GetDecoder()
		if err != nil {
			panic(err)
		}

		resource := &metav1.APIResourceList{}

		err = decoder.Decode(&resource)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break // End of resources
			}

			panic(fmt.Errorf("failed to decode resources: %w", err))
		}

		resources = append(resources, resource)
	}

	return resources
}

func getPreferredResources(group *metav1.APIGroup, resources []*metav1.APIResourceList) *metav1.APIResourceList {
	for _, r := range resources {
		if r.GroupVersion == group.PreferredVersion.GroupVersion {
			return r
		}
	}

	panic(fmt.Sprintf("preferred resources for group %#v not found", group.Name))
}
