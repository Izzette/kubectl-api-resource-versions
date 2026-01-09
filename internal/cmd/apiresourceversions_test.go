package cmd

import (
	"bytes"
	_ "embed"
	"errors"
	"io"
	"math/rand/v2"
	"reflect"
	"slices"
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

func TestExcludeGroupResource(t *testing.T) {
	t.Parallel()

	apiGroup := &metav1.APIGroup{
		Name: "apps",
		Versions: []metav1.GroupVersionForDiscovery{
			{GroupVersion: "apps/v1", Version: "v1"},
		},
		PreferredVersion: metav1.GroupVersionForDiscovery{GroupVersion: "apps/v1", Version: "v1"},
	}

	baseResource := groupResource{
		APIGroup:        apiGroup,
		APIGroupVersion: "apps/v1",
		APIResource: &metav1.APIResource{
			Name:       "deployments",
			Namespaced: true,
			Kind:       "Deployment",
		},
		Preferred:   true,
		Subresource: false,
	}

	subresource := groupResource{
		APIGroup:        apiGroup,
		APIGroupVersion: "apps/v1",
		APIResource: &metav1.APIResource{
			Name:       "deployments/status",
			Namespaced: true,
		},
		Preferred:   true,
		Subresource: true,
	}

	t.Run("BaseResourceIncludedByDefault", excludeGroupResourceTest{
		resource: baseResource,
		options:  NewTestOptionsBuilder().APIResourceVersionsOptions(),
		want:     false, // Should not be excluded.
	}.Test)
	t.Run("SubresourceExcludedByDefault", excludeGroupResourceTest{
		resource: subresource,
		options:  NewTestOptionsBuilder().APIResourceVersionsOptions(),
		want:     true, // Should be excluded because subresources are not included by default.
	}.Test)
	t.Run("SubresourceIncludedWithFlag", excludeGroupResourceTest{
		resource: subresource,
		options:  NewTestOptionsBuilder().SetIncludeSubresources(true).APIResourceVersionsOptions(),
		want:     false, // Should not be excluded when flag is set.
	}.Test)
	t.Run("BaseResourceIncludedWithFlag", excludeGroupResourceTest{
		resource: baseResource,
		options:  NewTestOptionsBuilder().SetIncludeSubresources(true).APIResourceVersionsOptions(),
		want:     false, // Should not be excluded.
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

func TestSplitResourceName(t *testing.T) {
	t.Parallel()

	nu := func(s string) *string { return &s }

	t.Run("ValidResourceNameWithGroupAndVersion", splitResourceNameTest{
		resourceName:     "deployments",
		baseResourceName: "deployments",
		subresourceName:  nil,
	}.Test)
	t.Run("ValidResourceNameWithSubresource", splitResourceNameTest{
		resourceName:     "deployments/status",
		baseResourceName: "deployments",
		subresourceName:  nu("status"),
	}.Test)
}

type splitResourceNameTest struct {
	resourceName     string
	baseResourceName string
	subresourceName  *string
}

func (tt splitResourceNameTest) Test(t *testing.T) {
	t.Parallel()

	gotBase, gotSub := splitResourceName(tt.resourceName)
	if gotBase != tt.baseResourceName {
		t.Errorf("splitResourceName() base = %v, want %v", gotBase, tt.baseResourceName)
	}

	switch {
	case tt.subresourceName == nil && gotSub != nil:
		t.Errorf("splitResourceName() expected nil subresource, got %v", *gotSub)
	case tt.subresourceName != nil && gotSub == nil:
		t.Errorf("splitResourceName() expected subresource %v, got nil", *tt.subresourceName)
	case tt.subresourceName != nil && gotSub != nil:
		if *gotSub != *tt.subresourceName {
			t.Errorf("splitResourceName() subresource = %v, want %v", *gotSub, *tt.subresourceName)
		}
	}
}

func TestUnversionedResourceName(t *testing.T) {
	t.Parallel()

	nu := func(s string) *string { return &s }

	t.Run("CoreResource", unversionedResourceNameTest{
		resource: metav1.APIResource{
			Name:       "pods",
			Namespaced: true,
			Version:    "v1",
			Kind:       "Pod",
		},
		baseResourceNameWithGroup: "pods.",
		subresourceName:           nil,
	}.Test)
	t.Run("NamedGroupResource", unversionedResourceNameTest{
		resource: metav1.APIResource{
			Name:       "deployments",
			Namespaced: true,
			Group:      "apps",
			Version:    "v1",
			Kind:       "Deployment",
		},
		baseResourceNameWithGroup: "deployments.apps",
		subresourceName:           nil,
	}.Test)
	t.Run("Subresource", unversionedResourceNameTest{
		resource: metav1.APIResource{
			Name:       "deployments/status",
			Namespaced: true,
			Group:      "apps",
			Version:    "v1",
		},
		baseResourceNameWithGroup: "deployments.apps",
		subresourceName:           nu("status"),
	}.Test)
}

type unversionedResourceNameTest struct {
	resource                  metav1.APIResource
	baseResourceNameWithGroup string
	subresourceName           *string
}

func (tt unversionedResourceNameTest) Test(t *testing.T) {
	t.Parallel()

	gotBase, gotSub := unversionedResourceName(tt.resource)
	if gotBase != tt.baseResourceNameWithGroup {
		t.Errorf("unversionedResourceName() base = %v, want %v", gotBase, tt.baseResourceNameWithGroup)
	}

	switch {
	case tt.subresourceName == nil && gotSub != nil:
		t.Errorf("unversionedResourceName() expected nil subresource, got %v", *gotSub)
	case tt.subresourceName != nil && gotSub == nil:
		t.Errorf("unversionedResourceName() expected subresource %v, got nil", *tt.subresourceName)
	case tt.subresourceName != nil && gotSub != nil:
		if *gotSub != *tt.subresourceName {
			t.Errorf("unversionedResourceName() subresource = %v, want %v", *gotSub, *tt.subresourceName)
		}
	}
}

func TestFullname(t *testing.T) {
	t.Parallel()

	apiGroup := &metav1.APIGroup{
		Name: "apps",
		Versions: []metav1.GroupVersionForDiscovery{
			{GroupVersion: "apps/v1", Version: "v1"},
		},
		PreferredVersion: metav1.GroupVersionForDiscovery{GroupVersion: "apps/v1", Version: "v1"},
	}

	coreAPIGroup := &metav1.APIGroup{
		Name: "",
		Versions: []metav1.GroupVersionForDiscovery{
			{GroupVersion: "v1", Version: "v1"},
		},
		PreferredVersion: metav1.GroupVersionForDiscovery{GroupVersion: "v1", Version: "v1"},
	}

	t.Run("BaseResourceWithGroup", fullnameTest{
		resource: groupResource{
			APIGroup:        apiGroup,
			APIGroupVersion: "apps/v1",
			APIResource: &metav1.APIResource{
				Name:       "deployments",
				Namespaced: true,
				Kind:       "Deployment",
			},
			Subresource: false,
		},
		want: "deployments.v1.apps",
	}.Test)
	t.Run("SubresourceWithGroup", fullnameTest{
		resource: groupResource{
			APIGroup:        apiGroup,
			APIGroupVersion: "apps/v1",
			APIResource: &metav1.APIResource{
				Name:       "deployments/status",
				Namespaced: true,
			},
			Subresource: true,
		},
		want: "deployments.v1.apps status",
	}.Test)
	t.Run("CoreResource", fullnameTest{
		resource: groupResource{
			APIGroup:        coreAPIGroup,
			APIGroupVersion: "v1",
			APIResource: &metav1.APIResource{
				Name:       "pods",
				Namespaced: true,
				Kind:       "Pod",
			},
			Subresource: false,
		},
		want: "pods.v1.",
	}.Test)
	t.Run("CoreSubresource", fullnameTest{
		resource: groupResource{
			APIGroup:        coreAPIGroup,
			APIGroupVersion: "v1",
			APIResource: &metav1.APIResource{
				Name:       "pods/status",
				Namespaced: true,
			},
			Subresource: true,
		},
		want: "pods.v1. status",
	}.Test)
}

type fullnameTest struct {
	resource groupResource
	want     string
}

func (tt fullnameTest) Test(t *testing.T) {
	t.Parallel()

	got := tt.resource.fullname()
	if got != tt.want {
		t.Errorf("fullname() = %v, want %v", got, tt.want)
	}
}

func TestGetPrefererredResourceVersions(t *testing.T) {
	t.Parallel()

	t.Run("GetPreferredVersions", getPreferredResourceVersionsTest{
		preferredVersions: []*metav1.APIResourceList{
			{
				GroupVersion: "apps/v1",
				APIResources: []metav1.APIResource{
					{Name: "deployments", Namespaced: true, Kind: "Deployment"},
					{Name: "deployments/status", Namespaced: true},
				},
			},
			{
				GroupVersion: "autoscaling/v2",
				APIResources: []metav1.APIResource{
					{Name: "horizontalpodautoscalers", Namespaced: true, Kind: "HorizontalPodAutoscaler"},
				},
			},
		},
		want: map[string]string{
			"deployments.apps":                     "v1",
			"horizontalpodautoscalers.autoscaling": "v2",
		},
		err: nil,
	}.Test)
}

type getPreferredResourceVersionsTest struct {
	preferredVersions []*metav1.APIResourceList
	want              map[string]string
	err               error
}

func (tt getPreferredResourceVersionsTest) Test(t *testing.T) {
	t.Parallel()

	builder := discoverytesting.NewFakeCachedDiscoveryClientBuilder()
	builder.PreferredResources = tt.preferredVersions
	options := NewTestOptionsBuilder().
		WithDiscoveryClient(builder.CachedDiscoveryInterface()).
		APIResourceVersionsOptions()

	got, err := getPreferredResourceVersions(options)
	if !errors.Is(err, tt.err) {
		t.Fatalf("getPreferredResourceVersions() error = %v, wantErr %v", err, tt.err)
	}

	if !reflect.DeepEqual(got, tt.want) {
		t.Errorf("getPreferredResourceVersions() = %v, want %v", got, tt.want)
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
	t.Run("GetAllWithSubresources", getGroupResourcesCountTest{
		options: NewTestOptionsBuilder().SetIncludeSubresources(true).APIResourceVersionsOptions(),
		// There are 13 base resources + 21 subresources in the test data.
		wantResourcesCount: 34,
	}.Test)
	t.Run("GetCoreSubresources", getGroupResourcesNamesTest{
		options: NewTestOptionsBuilder().
			SetAPIGroup("").
			SetIncludeSubresources(true).
			APIResourceVersionsOptions(),
		wantResourcesNames: []string{
			"configmaps.v1.",
			"events.v1.",
			"namespaces.v1.",
			"namespaces.v1. finalize",
			"namespaces.v1. status",
			"nodes.v1.",
			"nodes.v1. proxy",
			"nodes.v1. status",
			"persistentvolumeclaims.v1.",
			"persistentvolumeclaims.v1. status",
			"persistentvolumes.v1.",
			"persistentvolumes.v1. status",
			"pods.v1.",
			"pods.v1. attach",
			"pods.v1. binding",
			"pods.v1. ephemeralcontainers",
			"pods.v1. eviction",
			"pods.v1. exec",
			"pods.v1. log",
			"pods.v1. portforward",
			"pods.v1. proxy",
			"pods.v1. status",
			"secrets.v1.",
			"serviceaccounts.v1.",
			"serviceaccounts.v1. token",
			"services.v1.",
			"services.v1. proxy",
			"services.v1. status",
		},
	}.Test)
	t.Run("SubresourceExcludedByDefault", getGroupResourcesTest{
		options: NewTestOptionsBuilder().SetAPIGroup("").APIResourceVersionsOptions(),
		// Verify that resources like "pods/status" are not included.
		shouldNotContain: []string{"pods/status", "namespaces/status", "nodes/status"},
	}.Test)
	t.Run("SubresourceIncludedWithFlag", getGroupResourcesTest{
		options: NewTestOptionsBuilder().
			SetAPIGroup("").
			SetIncludeSubresources(true).
			APIResourceVersionsOptions(),
		// Verify that resources like "pods/status" are included.
		shouldContain: []string{"pods/status", "namespaces/status", "nodes/status"},
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

type getGroupResourcesTest struct {
	options          *apiResourceVersionsOptions
	shouldContain    []string
	shouldNotContain []string
	wantErr          error
}

func (tt getGroupResourcesTest) Test(t *testing.T) {
	t.Parallel()

	got, err := getGroupResources(tt.options)
	if !errors.Is(err, tt.wantErr) {
		t.Fatalf("getGroupResources() error = %v, wantErr %v", err, tt.wantErr)
	}

	gotNames := make([]string, len(got))
	for i, resource := range got {
		gotNames[i] = resource.APIResource.Name
	}

	for _, name := range tt.shouldContain {
		if !slices.Contains(gotNames, name) {
			t.Errorf("getGroupResources() should contain %q, but it was not found. Got: %v", name, gotNames)
		}
	}

	for _, name := range tt.shouldNotContain {
		for _, gotName := range gotNames {
			if gotName == name {
				t.Errorf("getGroupResources() should not contain %q, but it was found. Got: %v", name, gotNames)
			}
		}
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
		Preferred:   true,
		Subresource: false,
	}

	sampleSubresource := groupResource{
		APIGroup: &metav1.APIGroup{
			Name: "apps",
			PreferredVersion: metav1.GroupVersionForDiscovery{
				GroupVersion: "apps/v1",
				Version:      "v1",
			},
		},
		APIGroupVersion: "apps/v1",
		APIResource: &metav1.APIResource{
			Name:       "deployments/status",
			ShortNames: []string{},
			Namespaced: true,
			Kind:       "Deployment",
			Verbs:      []string{"get", "patch", "update"},
			Categories: []string{},
		},
		Preferred:   true,
		Subresource: true,
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
			want: "deployments  deploy  apps/v1  true  Deployment  true  " +
				"true  get,list,watch  all\n",
		},
		{
			name:     "name output",
			output:   nameOutput,
			resource: sampleResource,
			want:     "deployments.v1.apps\n",
		},
		{
			name:     "subresource default output",
			output:   "",
			resource: sampleSubresource,
			want:     "deployments/status    apps/v1  true  Deployment  true\n",
		},
		{
			name:     "subresource wide output",
			output:   wideOutput,
			resource: sampleSubresource,
			want: "deployments/status    apps/v1  true  Deployment  true  " +
				"true  get,patch,update  \n",
		},
		{
			name:     "subresource name output",
			output:   nameOutput,
			resource: sampleSubresource,
			want:     "deployments.v1.apps status\n",
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

			err := printFunc(writer, tt.resource)
			if err != nil {
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
		_, err := getGroupResources(options)
		if err != nil {
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
		err := printGroupResourcesDefault(io.Discard, groupResource)
		if err != nil {
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
