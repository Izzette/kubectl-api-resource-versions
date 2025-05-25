package cmd

import (
	"bytes"
	_ "embed"
	"errors"
	"io"
	"math/rand/v2"
	"reflect"
	"sort"
	"testing"

	"github.com/Izzette/kubectl-api-resource-versions/internal/discoverytesting"
	"github.com/liggitt/tabwriter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestValidateOptions tests validation of command options.
func TestValidateOptions(t *testing.T) {
	t.Parallel()

	t.Run("DefaultOptions", validateOptionsTest{
		options: NewTestOptionsBuilder().APIResourceVersionsOptions(),
		wantErr: nil,
	}.Test)
	t.Run("InvalidOutput", validateOptionsTest{
		options: NewTestOptionsBuilder().SetOutput("invalid").APIResourceVersionsOptions(),
		wantErr: errWrongOutput,
	}.Test)
	t.Run("InvalidSortBy", validateOptionsTest{
		options: NewTestOptionsBuilder().SetSortBy("invalid").APIResourceVersionsOptions(),
		wantErr: errSortBy,
	}.Test)
	t.Run("ValidSortByName", validateOptionsTest{
		options: NewTestOptionsBuilder().SetSortBy(nameSortBy).APIResourceVersionsOptions(),
		wantErr: nil,
	}.Test)
}

type validateOptionsTest struct {
	options *apiResourceVersionsOptions
	wantErr error
}

func (tt validateOptionsTest) Test(t *testing.T) {
	t.Parallel()

	err := tt.options.validate()
	if !errors.Is(err, tt.wantErr) {
		t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
	}
}

func TestExcludeGroup(t *testing.T) {
	t.Parallel()

	apiGroup := &metav1.APIGroup{
		Name: "apps",
		Versions: []metav1.GroupVersionForDiscovery{
			{GroupVersion: "apps/v1", Version: "v1"},
			{GroupVersion: "apps/v1beta1", Version: "v1beta1"},
		},
		PreferredVersion: metav1.GroupVersionForDiscovery{GroupVersion: "apps/v1", Version: "v1"},
	}

	t.Run("NoFilterMatch", excludeGroupTest{
		apiGroup: apiGroup,
		options:  NewTestOptionsBuilder().APIResourceVersionsOptions(),
		want:     false, // Should not be excluded because no filters are applied.
	}.Test)
	t.Run("FilterMatch", excludeGroupTest{
		apiGroup: apiGroup,
		options:  NewTestOptionsBuilder().SetAPIGroup("apps").APIResourceVersionsOptions(),
		want:     false, // Should not be excluded because the group matches.
	}.Test)
	t.Run("FilterNoMatch", excludeGroupTest{
		apiGroup: apiGroup,
		options:  NewTestOptionsBuilder().SetAPIGroup("nonexistent").APIResourceVersionsOptions(),
		want:     true, // Should be excluded because the group does not match.
	}.Test)
}

type excludeGroupTest struct {
	apiGroup *metav1.APIGroup
	options  *apiResourceVersionsOptions
	want     bool
}

func (tt excludeGroupTest) Test(t *testing.T) {
	t.Parallel()

	got := excludeGroup(tt.apiGroup, tt.options)
	if got != tt.want {
		t.Errorf("excludeGroup() = %v, want %v", got, tt.want)
	}
}

func TestExcludeGroupVersion(t *testing.T) {
	t.Parallel()

	apiGroup := &metav1.APIGroup{
		Name: "apps",
		Versions: []metav1.GroupVersionForDiscovery{
			{GroupVersion: "apps/v1", Version: "v1"},
			{GroupVersion: "apps/v1beta1", Version: "v1beta1"},
		},
		PreferredVersion: metav1.GroupVersionForDiscovery{GroupVersion: "apps/v1", Version: "v1"},
	}

	t.Run("NoFilterMatch", excludeGroupVersionTest{
		apiGroup:        apiGroup,
		apiGroupVersion: "apps/v1",
		options:         NewTestOptionsBuilder().APIResourceVersionsOptions(),
		want:            false, // Should not be excluded because no filters are applied.
	}.Test)
	t.Run("PreferredFilterMatch", excludeGroupVersionTest{
		apiGroup:        apiGroup,
		apiGroupVersion: "apps/v1",
		options:         NewTestOptionsBuilder().SetPreferred(true).APIResourceVersionsOptions(),
		want:            false, // Should not be excluded because the group matches.
	}.Test)
	t.Run("NonPreferredFilterMatch", excludeGroupVersionTest{
		apiGroup:        apiGroup,
		apiGroupVersion: "apps/v1beta1",
		options:         NewTestOptionsBuilder().SetPreferred(false).APIResourceVersionsOptions(),
		want:            false, // Should be not be excluded because the group is not preferred.
	}.Test)
	t.Run("PreferredFilterNoMatch", excludeGroupVersionTest{
		apiGroup:        apiGroup,
		apiGroupVersion: "apps/v1beta1",
		options:         NewTestOptionsBuilder().SetPreferred(true).APIResourceVersionsOptions(),
		want:            true, // Should be excluded because the group is preferred but the version is not.
	}.Test)
	t.Run("NonPreferredFilterNoMatch", excludeGroupVersionTest{
		apiGroup:        apiGroup,
		apiGroupVersion: "apps/v1",
		options:         NewTestOptionsBuilder().SetPreferred(false).APIResourceVersionsOptions(),
		want:            true, // Should be excluded because the group is not preferred.
	}.Test)
}

type excludeGroupVersionTest struct {
	apiGroup        *metav1.APIGroup
	apiGroupVersion string
	options         *apiResourceVersionsOptions
	want            bool
}

func (tt excludeGroupVersionTest) Test(t *testing.T) {
	t.Parallel()

	got := excludeGroupVersion(tt.apiGroup, tt.apiGroupVersion, tt.options)
	if got != tt.want {
		t.Errorf("excludeGroupVersion() = %v, want %v", got, tt.want)
	}
}

// TestExcludeGroupResource tests resource filtering logic.
func TestExcludeGroupResource(t *testing.T) {
	t.Parallel()

	baseResource := groupResource{
		APIGroup: &metav1.APIGroup{
			Name: "apps",
			Versions: []metav1.GroupVersionForDiscovery{
				{GroupVersion: "apps/v1", Version: "v1"},
				{GroupVersion: "apps/v1beta1", Version: "v1beta1"},
			},
			PreferredVersion: metav1.GroupVersionForDiscovery{GroupVersion: "apps/v1", Version: "v1"},
		},
		APIGroupVersion: "apps/v1",
		APIResource: &metav1.APIResource{
			Name:       "deployments",
			Namespaced: true,
			Verbs:      []string{"get", "list", "watch"},
			Categories: []string{"all"},
		},
	}

	t.Run("NoFilterMatch", excludeGroupResourceTest{
		resource: baseResource,
		options:  NewTestOptionsBuilder().APIResourceVersionsOptions(),
		want:     false, // Should not be excluded because no filters are applied.
	}.Test)
	t.Run("VerbFilterMatch", excludeGroupResourceTest{
		resource: baseResource,
		options:  NewTestOptionsBuilder().SetVerbs([]string{"get"}).APIResourceVersionsOptions(),
		want:     false, // Should not be excluded because "get" is a valid verb.
	}.Test)
	t.Run("VerbFilterNoMatch", excludeGroupResourceTest{
		resource: baseResource,
		options:  NewTestOptionsBuilder().SetVerbs([]string{"create"}).APIResourceVersionsOptions(),
		want:     true, // Should be excluded because "create" is not a valid verb.
	}.Test)
	t.Run("CategoryFilterMatch", excludeGroupResourceTest{
		resource: baseResource,
		options:  NewTestOptionsBuilder().SetCategories([]string{"all"}).APIResourceVersionsOptions(),
		want:     false, // Should not be excluded because "all" is a valid category.
	}.Test)
	t.Run("CategoryFilterNoMatch", excludeGroupResourceTest{
		resource: baseResource,
		options:  NewTestOptionsBuilder().SetCategories([]string{"custom"}).APIResourceVersionsOptions(),
		want:     true, // Should be excluded because "custom" is not a valid category.
	}.Test)
	t.Run("NamespacedFilterMatch", excludeGroupResourceTest{
		resource: baseResource,
		options:  NewTestOptionsBuilder().SetNamespaced(true).APIResourceVersionsOptions(),
		want:     false, // Should not be excluded because the resource is namespaced.
	}.Test)
	t.Run("NamespacedFilterNoMatch", excludeGroupResourceTest{
		resource: baseResource,
		options:  NewTestOptionsBuilder().SetNamespaced(false).APIResourceVersionsOptions(),
		want:     true, // Should be excluded because the resource is namespaced but filter is for non-namespaced.
	}.Test)
	t.Run("PreferredFilterMatch", excludeGroupResourceTest{
		resource: baseResource,
		options:  NewTestOptionsBuilder().SetPreferred(true).APIResourceVersionsOptions(),
		want:     false, // Should not be excluded because the resource is preferred.
	}.Test)
	t.Run("PreferredFilterNoMatch", excludeGroupResourceTest{
		resource: baseResource,
		options:  NewTestOptionsBuilder().SetPreferred(false).APIResourceVersionsOptions(),
		want:     true, // Should be excluded because the resource is preferred but filter is for non-preferred.
	}.Test)
}

type excludeGroupResourceTest struct {
	resource groupResource
	options  *apiResourceVersionsOptions
	want     bool
}

func (tt excludeGroupResourceTest) Test(t *testing.T) {
	t.Parallel()

	got := excludeGroupResource(tt.resource, tt.options)
	if got != tt.want {
		t.Errorf("excludeGroupResource() = %v, want %v", got, tt.want)
	}
}

// TestGetGroupResources tests resource discovery and processing.
func TestGetGroupResources(t *testing.T) {
	t.Parallel()

	t.Run("GetAll", getGroupResourcesCountTest{
		options:            NewTestOptionsBuilder().APIResourceVersionsOptions(),
		wantResourcesCount: 13, // There are 13 non-subresource resources in the test data.
	}.Test)
	t.Run("GetCoreNonNamespaced", getGroupResourcesNamesTest{
		options: NewTestOptionsBuilder().SetAPIGroup("").SetNamespaced(false).APIResourceVersionsOptions(),
		wantResourcesNames: []string{
			"namespaces.v1.",
			"nodes.v1.",
			"persistentvolumes.v1.",
		},
	}.Test)
	t.Run("GetAutoscalingNamespaced", getGroupResourcesNamesTest{
		options: NewTestOptionsBuilder().SetAPIGroup("autoscaling").SetNamespaced(true).APIResourceVersionsOptions(),
		wantResourcesNames: []string{
			"horizontalpodautoscalers.v2.autoscaling",
			"horizontalpodautoscalers.v1.autoscaling",
			"horizontalpodautoscalers.v2beta2.autoscaling",
		},
		wantErr: nil,
	}.Test)
	t.Run("GetAutoscalingNonNamespaced", getGroupResourcesCountTest{
		options: NewTestOptionsBuilder().
			SetAPIGroup("autoscaling").SetNamespaced(false).APIResourceVersionsOptions(),
		wantResourcesCount: 0, // There are no non-namespaced resources in the autoscaling group.
	}.Test)
	t.Run("GetCoreDeleteCollection", getGroupResourcesNamesTest{
		options: NewTestOptionsBuilder().SetAPIGroup("").SetVerbs([]string{"deletecollection"}).APIResourceVersionsOptions(),
		wantResourcesNames: []string{
			"configmaps.v1.",
			"events.v1.",
			"nodes.v1.",
			"persistentvolumeclaims.v1.",
			"persistentvolumes.v1.",
			"pods.v1.",
			"secrets.v1.",
			"serviceaccounts.v1.",
			"services.v1.",
		},
	}.Test)
	t.Run("GetAutoscalingPreferredVersions", getGroupResourcesNamesTest{
		options: NewTestOptionsBuilder().SetAPIGroup("autoscaling").SetPreferred(true).APIResourceVersionsOptions(),
		wantResourcesNames: []string{
			"horizontalpodautoscalers.v2.autoscaling",
		},
		wantErr: nil,
	}.Test)
	t.Run("GetAutoscalingNonPreferredVersions", getGroupResourcesNamesTest{
		options: NewTestOptionsBuilder().SetAPIGroup("autoscaling").SetPreferred(false).APIResourceVersionsOptions(),
		wantResourcesNames: []string{
			"horizontalpodautoscalers.v1.autoscaling",
			"horizontalpodautoscalers.v2beta2.autoscaling",
		},
		wantErr: nil,
	}.Test)
	t.Run("GetNoResources", getGroupResourcesCountTest{
		options:            NewTestOptionsBuilder().SetAPIGroup("nonexistent").APIResourceVersionsOptions(),
		wantResourcesCount: 0,   // No resources should be found for a non-existent group.
		wantErr:            nil, // No error expected, just an empty result.
	}.Test)
}

type getGroupResourcesCountTest struct {
	options            *apiResourceVersionsOptions
	wantResourcesCount int
	wantErr            error
}

func (tt getGroupResourcesCountTest) Test(t *testing.T) {
	t.Parallel()

	got, err := getGroupResources(tt.options)
	if !errors.Is(err, tt.wantErr) {
		t.Fatalf("getGroupResources() error = %v, wantErr %v", err, tt.wantErr)
	}
	if len(got) != tt.wantResourcesCount {
		t.Errorf("getGroupResources() count = %d, want %d", len(got), tt.wantResourcesCount)
	}
}

type getGroupResourcesNamesTest struct {
	options            *apiResourceVersionsOptions
	wantResourcesNames []string
	wantErr            error
}

func (tt getGroupResourcesNamesTest) Test(t *testing.T) {
	t.Parallel()

	got, err := getGroupResources(tt.options)
	if !errors.Is(err, tt.wantErr) {
		t.Fatalf("getGroupResources() error = %v, wantErr %v", err, tt.wantErr)
	}
	if len(got) != len(tt.wantResourcesNames) {
		t.Errorf("getGroupResources() count = %d, want %d", len(got), len(tt.wantResourcesNames))
	}
	gotNames := make([]string, len(got))
	for i, resource := range got {
		gotNames[i] = resource.fullname()
	}
	if !reflect.DeepEqual(gotNames, tt.wantResourcesNames) {
		t.Errorf("getGroupResources() names = %v, want %v", gotNames, tt.wantResourcesNames)
	}
}

// TestPrintFunctions tests output formatting.
func TestPrintFunctions(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		output   string
		resource groupResource
		want     string
	}

	sampleResource := groupResource{
		APIGroup: &metav1.APIGroup{
			Name: "apps",
			PreferredVersion: metav1.GroupVersionForDiscovery{
				GroupVersion: "apps/v1",
				Version:      "v1",
			},
		},
		APIGroupVersion: "apps/v1",
		APIResource: &metav1.APIResource{
			Name:         "deployments",
			SingularName: "deployment",
			ShortNames:   []string{"deploy"},
			Namespaced:   true,
			Kind:         "Deployment",
			Verbs:        []string{"get", "list", "watch"},
			Categories:   []string{"all"},
		},
	}

	tests := []testCase{
		{
			name:     "default output",
			output:   "",
			resource: sampleResource,
			want:     "deployments  deploy  apps/v1  true  Deployment  true\n",
		},
		{
			name:     "wide output",
			output:   wideOutput,
			resource: sampleResource,
			want:     "deployments  deploy  apps/v1  true  Deployment  true  get,list,watch  all\n",
		},
		{
			name:     "name output",
			output:   nameOutput,
			resource: sampleResource,
			want:     "deployments.v1.apps\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			buf := new(bytes.Buffer)
			writer := tabwriter.NewWriter(buf, 0, 0, 2, ' ', 0)

			var printFunc func(io.Writer, groupResource) error
			switch tt.output {
			case wideOutput:
				printFunc = printGroupResourcesWide
			case nameOutput:
				printFunc = printGroupResourcesByName
			default:
				printFunc = printGroupResourcesDefault
			}

			if err := printFunc(writer, tt.resource); err != nil {
				t.Fatalf("print function failed: %v", err)
			}
			mustFlushWriter(writer)

			if buf.String() != tt.want {
				t.Errorf("print output = %q, want %q", buf.String(), tt.want)
			}
		})
	}
}

// TestSorting tests resource sorting logic.
func TestSorting(t *testing.T) {
	t.Parallel()

	resources := []groupResource{
		{APIResource: &metav1.APIResource{Name: "b-kind", Kind: "BKind"}, APIGroup: &metav1.APIGroup{Name: "foo"}},
		{APIResource: &metav1.APIResource{Name: "z-kind", Kind: "AKind"}, APIGroup: &metav1.APIGroup{Name: "foo"}},
		{APIResource: &metav1.APIResource{Name: "m-kind", Kind: "CKind"}, APIGroup: &metav1.APIGroup{Name: "bar"}},
	}

	t.Run("sort by name", func(t *testing.T) {
		t.Parallel()

		resourcesCopy := make([]groupResource, len(resources))
		copy(resourcesCopy, resources)
		sorter := sortableResource{resources: resourcesCopy, sortBy: nameSortBy}
		sort.Sort(sorter)
		if sorter.resources[0].APIResource.Name != "b-kind" {
			t.Error("resources not sorted by resource name")
		}
	})

	t.Run("sort by kind", func(t *testing.T) {
		t.Parallel()

		resourcesCopy := make([]groupResource, len(resources))
		copy(resourcesCopy, resources)
		sorter := sortableResource{resources: resourcesCopy, sortBy: kindSortBy}
		sort.Sort(sorter)
		if sorter.resources[0].APIResource.Kind != "AKind" {
			t.Error("resources not sorted by kind")
		}
	})

	t.Run("sort by default", func(t *testing.T) {
		t.Parallel()

		resourcesCopy := make([]groupResource, len(resources))
		copy(resourcesCopy, resources)
		sorter := sortableResource{resources: resourcesCopy}
		sort.Sort(sorter)
		if sorter.resources[0].APIGroup.Name != "bar" {
			t.Error("resources not sorted by api group name")
		}
		if sorter.resources[1].APIResource.Name != "b-kind" {
			t.Error("resources not sorted by resource name")
		}
	})
}

// BenchmarkGet3000GroupResources benchmarks resource discovery performance
// It filters and enumerates 3000 resources across 100 groups and 3 versions each.
func BenchmarkGet3000GroupResources(b *testing.B) {
	// Create a large fake discovery client
	cached := discoverytesting.NewProcedural(100, 3, 30)
	options := NewTestOptionsBuilder().WithDiscoveryClient(cached).APIResourceVersionsOptions()

	for b.Loop() {
		if _, err := getGroupResources(options); err != nil {
			b.Fatalf("getGroupResources failed: %v", err)
		}
	}
}

// BenchmarkPrintGroupResources benchmarks the performance of printing group resources using the default print function.
func BenchmarkPrintGroupResources(b *testing.B) {
	// Create a large fake discovery client
	groupResource := groupResource{
		APIGroup: &metav1.APIGroup{
			Name: "testgroup",
			Versions: []metav1.GroupVersionForDiscovery{
				{GroupVersion: "testgroup/v1", Version: "v1"},
				{GroupVersion: "testgroup/v2", Version: "v2"},
			},
			PreferredVersion: metav1.GroupVersionForDiscovery{GroupVersion: "testgroup/v1", Version: "v1"},
		},
		APIGroupVersion: "testgroup/v1",
		APIResource: &metav1.APIResource{
			Name:         "testresource",
			SingularName: "testresource",
			ShortNames:   []string{"tr"},
			Namespaced:   true,
			Kind:         "TestResource",
			Verbs:        []string{"get", "list", "watch"},
			Categories:   []string{"all"},
		},
	}

	for b.Loop() {
		// Print the resources using the default print function
		if err := printGroupResourcesDefault(io.Discard, groupResource); err != nil {
			b.Fatalf("printGroupResourcesDefault failed: %v", err)
		}
	}
}

func BenchmarkSortableResource(b *testing.B) {
	cached := discoverytesting.NewProcedural(100, 3, 30)
	options := NewTestOptionsBuilder().WithDiscoveryClient(cached).APIResourceVersionsOptions()
	// Create a large number of resources for sorting
	groupResources, err := getGroupResources(options)
	if err != nil {
		b.Fatalf("getGroupResources failed: %v", err)
	}

	// Shuffle the resources to ensure randomness
	source := rand.NewPCG(0x4594815ce8d2f9, 0x22fad47d537bd2bd)
	r := rand.New(source) // Set a fixed seed for reproducibility
	r.Shuffle(len(groupResources), func(i, j int) {
		groupResources[i], groupResources[j] = groupResources[j], groupResources[i]
	})

	b.ResetTimer()
	for b.Loop() {
		// Create a sortableResource instance with the shuffled resources
		copyOfGroupResources := make([]groupResource, len(groupResources))
		copy(copyOfGroupResources, groupResources)
		sortable := sortableResource{
			resources: copyOfGroupResources,
			sortBy:    "", // Sort by the default criteria
		}

		sort.Sort(sortable)
		if len(sortable.resources) == 0 {
			b.Fatal("no resources to sort")
		}
	}
}
