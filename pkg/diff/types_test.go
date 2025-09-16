package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResults_FilterByType(t *testing.T) {
	results := Results{
		ResourceKey{Kind: "Deployment", Name: "changed-app"}:   {Type: Changed, Diff: "changed diff"},
		ResourceKey{Kind: "Service", Name: "created-service"}:  {Type: Created, Diff: "created diff"},
		ResourceKey{Kind: "ConfigMap", Name: "deleted-config"}: {Type: Deleted, Diff: "deleted diff"},
		ResourceKey{Kind: "Secret", Name: "unchanged-secret"}:  {Type: Unchanged, Diff: ""},
	}

	tests := []struct {
		name          string
		changeType    ChangeType
		expectedCount int
		expectedKeys  []ResourceKey
	}{
		{
			name:          "filter by Changed type",
			changeType:    Changed,
			expectedCount: 1,
			expectedKeys:  []ResourceKey{{Kind: "Deployment", Name: "changed-app"}},
		},
		{
			name:          "filter by Created type",
			changeType:    Created,
			expectedCount: 1,
			expectedKeys:  []ResourceKey{{Kind: "Service", Name: "created-service"}},
		},
		{
			name:          "filter by Deleted type",
			changeType:    Deleted,
			expectedCount: 1,
			expectedKeys:  []ResourceKey{{Kind: "ConfigMap", Name: "deleted-config"}},
		},
		{
			name:          "filter by Unchanged type",
			changeType:    Unchanged,
			expectedCount: 1,
			expectedKeys:  []ResourceKey{{Kind: "Secret", Name: "unchanged-secret"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := results.FilterByType(tt.changeType)
			assert.Equal(t, tt.expectedCount, len(filtered))

			for _, expectedKey := range tt.expectedKeys {
				assert.Contains(t, filtered, expectedKey)
			}
		})
	}

	t.Run("convenience methods match FilterByType", func(t *testing.T) {
		assert.Equal(t, results.FilterByType(Changed), results.FilterChanged())
		assert.Equal(t, results.FilterByType(Created), results.FilterCreated())
		assert.Equal(t, results.FilterByType(Deleted), results.FilterDeleted())
		assert.Equal(t, results.FilterByType(Unchanged), results.FilterUnchanged())
	})
}

func TestResults_FilterByAttributes(t *testing.T) {
	results := Results{
		ResourceKey{Kind: "Deployment", Namespace: "default", Name: "app1"}:    {Type: Changed, Diff: "diff1"},
		ResourceKey{Kind: "Service", Namespace: "default", Name: "app1"}:       {Type: Created, Diff: "diff2"},
		ResourceKey{Kind: "Deployment", Namespace: "production", Name: "app2"}: {Type: Deleted, Diff: "diff3"},
		ResourceKey{Kind: "ConfigMap", Namespace: "default", Name: "config"}:   {Type: Unchanged, Diff: ""},
	}

	tests := []struct {
		name          string
		filterFunc    func(Results) Results
		expectedCount int
		expectedKeys  []ResourceKey
	}{
		{
			name:          "filter by Kind - Deployment",
			filterFunc:    func(r Results) Results { return r.FilterByKind("Deployment") },
			expectedCount: 2,
			expectedKeys: []ResourceKey{
				{Kind: "Deployment", Namespace: "default", Name: "app1"},
				{Kind: "Deployment", Namespace: "production", Name: "app2"},
			},
		},
		{
			name:          "filter by Namespace - default",
			filterFunc:    func(r Results) Results { return r.FilterByNamespace("default") },
			expectedCount: 3,
			expectedKeys: []ResourceKey{
				{Kind: "Deployment", Namespace: "default", Name: "app1"},
				{Kind: "Service", Namespace: "default", Name: "app1"},
				{Kind: "ConfigMap", Namespace: "default", Name: "config"},
			},
		},
		{
			name:          "filter by ResourceName - app1",
			filterFunc:    func(r Results) Results { return r.FilterByResourceName("app1") },
			expectedCount: 2,
			expectedKeys: []ResourceKey{
				{Kind: "Deployment", Namespace: "default", Name: "app1"},
				{Kind: "Service", Namespace: "default", Name: "app1"},
			},
		},
		{
			name:          "chained filters - default namespace Deployments",
			filterFunc:    func(r Results) Results { return r.FilterByNamespace("default").FilterByKind("Deployment") },
			expectedCount: 1,
			expectedKeys:  []ResourceKey{{Kind: "Deployment", Namespace: "default", Name: "app1"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := tt.filterFunc(results)
			assert.Equal(t, tt.expectedCount, len(filtered))

			for _, expectedKey := range tt.expectedKeys {
				assert.Contains(t, filtered, expectedKey)
			}
		})
	}
}

func TestResults_Apply(t *testing.T) {
	results := Results{
		ResourceKey{Kind: "Deployment", Namespace: "default", Name: "app1"}:    {Type: Changed, Diff: "diff1"},
		ResourceKey{Kind: "Service", Namespace: "default", Name: "app1"}:       {Type: Created, Diff: "diff2"},
		ResourceKey{Kind: "Deployment", Namespace: "production", Name: "app2"}: {Type: Deleted, Diff: "diff3"},
		ResourceKey{Kind: "ConfigMap", Namespace: "default", Name: "config"}:   {Type: Unchanged, Diff: ""},
	}

	tests := []struct {
		name          string
		filterFunc    func(key ResourceKey, result Result) bool
		expectedCount int
		expectedKeys  []ResourceKey
	}{
		{
			name: "default namespace with changes",
			filterFunc: func(key ResourceKey, result Result) bool {
				return key.Namespace == "default" && result.Type != Unchanged
			},
			expectedCount: 2,
			expectedKeys: []ResourceKey{
				{Kind: "Deployment", Namespace: "default", Name: "app1"},
				{Kind: "Service", Namespace: "default", Name: "app1"},
			},
		},
		{
			name: "production namespace resources",
			filterFunc: func(key ResourceKey, _ Result) bool {
				return key.Namespace == "production"
			},
			expectedCount: 1,
			expectedKeys:  []ResourceKey{{Kind: "Deployment", Namespace: "production", Name: "app2"}},
		},
		{
			name: "only unchanged resources",
			filterFunc: func(_ ResourceKey, result Result) bool {
				return result.Type == Unchanged
			},
			expectedCount: 1,
			expectedKeys:  []ResourceKey{{Kind: "ConfigMap", Namespace: "default", Name: "config"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := results.Apply(tt.filterFunc)
			assert.Equal(t, tt.expectedCount, len(filtered))

			for _, expectedKey := range tt.expectedKeys {
				assert.Contains(t, filtered, expectedKey)
			}
		})
	}
}

func TestResults_Analysis(t *testing.T) {
	results := Results{
		ResourceKey{Kind: "Deployment", Name: "changed-app"}:   {Type: Changed, Diff: "changed diff"},
		ResourceKey{Kind: "Service", Name: "created-service"}:  {Type: Created, Diff: "created diff"},
		ResourceKey{Kind: "ConfigMap", Name: "deleted-config"}: {Type: Deleted, Diff: "deleted diff"},
		ResourceKey{Kind: "Secret", Name: "unchanged-secret"}:  {Type: Unchanged, Diff: ""},
	}

	noChangesResults := Results{
		ResourceKey{Kind: "Secret", Name: "unchanged-secret"}: {Type: Unchanged, Diff: ""},
	}

	emptyResults := Results{}

	tests := []struct {
		name                string
		results             Results
		expectedHasChanges  bool
		expectedIsEmpty     bool
		expectedCount       int
		expectedCountByType map[ChangeType]int
	}{
		{
			name:               "results with changes",
			results:            results,
			expectedHasChanges: true,
			expectedIsEmpty:    false,
			expectedCount:      4,
			expectedCountByType: map[ChangeType]int{
				Changed:   1,
				Created:   1,
				Deleted:   1,
				Unchanged: 1,
			},
		},
		{
			name:               "results without changes",
			results:            noChangesResults,
			expectedHasChanges: false,
			expectedIsEmpty:    false,
			expectedCount:      1,
			expectedCountByType: map[ChangeType]int{
				Changed:   0,
				Created:   0,
				Deleted:   0,
				Unchanged: 1,
			},
		},
		{
			name:               "empty results",
			results:            emptyResults,
			expectedHasChanges: false,
			expectedIsEmpty:    true,
			expectedCount:      0,
			expectedCountByType: map[ChangeType]int{
				Changed:   0,
				Created:   0,
				Deleted:   0,
				Unchanged: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedHasChanges, tt.results.HasChanges())
			assert.Equal(t, tt.expectedIsEmpty, tt.results.IsEmpty())
			assert.Equal(t, tt.expectedCount, tt.results.Count())

			for changeType, expectedCount := range tt.expectedCountByType {
				assert.Equal(t, expectedCount, tt.results.CountByType(changeType))
			}

			keys := tt.results.GetResourceKeys()
			assert.Equal(t, tt.expectedCount, len(keys))
		})
	}

	t.Run("GetResourceKeysByType", func(t *testing.T) {
		changedKeys := results.GetResourceKeysByType(Changed)
		assert.Equal(t, 1, len(changedKeys))
		assert.Equal(t, "changed-app", changedKeys[0].Name)

		createdKeys := results.GetResourceKeysByType(Created)
		assert.Equal(t, 1, len(createdKeys))
		assert.Equal(t, "created-service", createdKeys[0].Name)

		invalidKeys := results.GetResourceKeysByType(ChangeType(99))
		assert.Equal(t, 0, len(invalidKeys))
	})
}

func TestResults_GetStatistics(t *testing.T) {
	tests := []struct {
		name              string
		results           Results
		expectedTotal     int
		expectedChanged   int
		expectedCreated   int
		expectedDeleted   int
		expectedUnchanged int
	}{
		{
			name: "mixed results",
			results: Results{
				ResourceKey{Kind: "Deployment", Name: "app1"}:  {Type: Changed, Diff: "diff1"},
				ResourceKey{Kind: "Deployment", Name: "app2"}:  {Type: Changed, Diff: "diff2"},
				ResourceKey{Kind: "Service", Name: "svc1"}:     {Type: Created, Diff: "diff3"},
				ResourceKey{Kind: "ConfigMap", Name: "config"}: {Type: Deleted, Diff: "diff4"},
				ResourceKey{Kind: "Secret", Name: "secret1"}:   {Type: Unchanged, Diff: ""},
				ResourceKey{Kind: "Secret", Name: "secret2"}:   {Type: Unchanged, Diff: ""},
			},
			expectedTotal:     6,
			expectedChanged:   2,
			expectedCreated:   1,
			expectedDeleted:   1,
			expectedUnchanged: 2,
		},
		{
			name:              "empty results",
			results:           Results{},
			expectedTotal:     0,
			expectedChanged:   0,
			expectedCreated:   0,
			expectedDeleted:   0,
			expectedUnchanged: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := tt.results.GetStatistics()

			assert.Equal(t, tt.expectedTotal, stats.Total)
			assert.Equal(t, tt.expectedChanged, stats.Changed)
			assert.Equal(t, tt.expectedCreated, stats.Created)
			assert.Equal(t, tt.expectedDeleted, stats.Deleted)
			assert.Equal(t, tt.expectedUnchanged, stats.Unchanged)
		})
	}
}

func TestResults_StringSummary(t *testing.T) {
	results := Results{
		ResourceKey{Kind: "Deployment", Namespace: "default", Name: "app1"}:    {Type: Changed, Diff: "diff1"},
		ResourceKey{Kind: "Deployment", Namespace: "production", Name: "app2"}: {Type: Changed, Diff: "diff2"},
		ResourceKey{Kind: "Service", Namespace: "default", Name: "svc1"}:       {Type: Created, Diff: "diff3"},
		ResourceKey{Kind: "ConfigMap", Name: "config1"}:                        {Type: Deleted, Diff: "diff4"}, // cluster-scoped
		ResourceKey{Kind: "Secret", Namespace: "default", Name: "secret1"}:     {Type: Unchanged, Diff: ""},
	}

	unchangedOnlyResults := Results{
		ResourceKey{Kind: "Secret", Namespace: "default", Name: "secret1"}: {Type: Unchanged, Diff: ""},
	}

	emptyResults := Results{}

	tests := []struct {
		name             string
		results          Results
		shouldContain    []string
		shouldNotContain []string
		expectEmpty      bool
	}{
		{
			name:    "mixed results summary",
			results: results,
			shouldContain: []string{
				"Unchanged (1):", "Changed (2):", "Create (1):", "Delete (1):",
				"Secret/default/secret1",
				"Deployment/default/app1",
				"Deployment/production/app2",
				"Service/default/svc1",
				"ConfigMap/config1",
			},
			shouldNotContain: []string{},
			expectEmpty:      false,
		},
		{
			name:    "unchanged only summary",
			results: unchangedOnlyResults,
			shouldContain: []string{
				"Unchanged (1):",
				"Secret/default/secret1",
			},
			shouldNotContain: []string{
				"Changed:", "Create:", "Delete:",
			},
			expectEmpty: false,
		},
		{
			name:        "empty results summary",
			results:     emptyResults,
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := tt.results.StringSummary()

			if tt.expectEmpty {
				assert.Equal(t, "", summary)
				return
			}

			for _, expected := range tt.shouldContain {
				assert.Contains(t, summary, expected)
			}

			for _, notExpected := range tt.shouldNotContain {
				assert.NotContains(t, summary, notExpected)
			}
		})
	}
}
