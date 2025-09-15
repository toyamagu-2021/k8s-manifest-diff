package diff

import (
	"testing"

	"github.com/argoproj/gitops-engine/pkg/utils/kube"
	"github.com/stretchr/testify/assert"
)

func TestResults_FilterByType(t *testing.T) {
	// Create test results with various change types
	results := Results{
		kube.ResourceKey{Kind: "Deployment", Name: "changed-app"}:   {Type: Changed, Diff: "changed diff"},
		kube.ResourceKey{Kind: "Service", Name: "created-service"}:  {Type: Created, Diff: "created diff"},
		kube.ResourceKey{Kind: "ConfigMap", Name: "deleted-config"}: {Type: Deleted, Diff: "deleted diff"},
		kube.ResourceKey{Kind: "Secret", Name: "unchanged-secret"}:  {Type: Unchanged, Diff: ""},
	}

	// Test FilterByType
	changedResults := results.FilterByType(Changed)
	assert.Equal(t, 1, len(changedResults))
	assert.Contains(t, changedResults, kube.ResourceKey{Kind: "Deployment", Name: "changed-app"})

	createdResults := results.FilterByType(Created)
	assert.Equal(t, 1, len(createdResults))
	assert.Contains(t, createdResults, kube.ResourceKey{Kind: "Service", Name: "created-service"})

	// Test convenience methods
	assert.Equal(t, changedResults, results.FilterChanged())
	assert.Equal(t, createdResults, results.FilterCreated())
	assert.Equal(t, results.FilterByType(Deleted), results.FilterDeleted())
	assert.Equal(t, results.FilterByType(Unchanged), results.FilterUnchanged())
}

func TestResults_FilterByAttributes(t *testing.T) {
	results := Results{
		kube.ResourceKey{Kind: "Deployment", Namespace: "default", Name: "app1"}:    {Type: Changed, Diff: "diff1"},
		kube.ResourceKey{Kind: "Service", Namespace: "default", Name: "app1"}:       {Type: Created, Diff: "diff2"},
		kube.ResourceKey{Kind: "Deployment", Namespace: "production", Name: "app2"}: {Type: Deleted, Diff: "diff3"},
		kube.ResourceKey{Kind: "ConfigMap", Namespace: "default", Name: "config"}:   {Type: Unchanged, Diff: ""},
	}

	// Test FilterByKind
	deployments := results.FilterByKind("Deployment")
	assert.Equal(t, 2, len(deployments))

	// Test FilterByNamespace
	defaultNS := results.FilterByNamespace("default")
	assert.Equal(t, 3, len(defaultNS))

	// Test FilterByResourceName
	app1Resources := results.FilterByResourceName("app1")
	assert.Equal(t, 2, len(app1Resources))

	// Test chaining filters
	defaultDeployments := results.FilterByNamespace("default").FilterByKind("Deployment")
	assert.Equal(t, 1, len(defaultDeployments))
	assert.Contains(t, defaultDeployments, kube.ResourceKey{Kind: "Deployment", Namespace: "default", Name: "app1"})
}

func TestResults_Apply(t *testing.T) {
	results := Results{
		kube.ResourceKey{Kind: "Deployment", Namespace: "default", Name: "app1"}:    {Type: Changed, Diff: "diff1"},
		kube.ResourceKey{Kind: "Service", Namespace: "default", Name: "app1"}:       {Type: Created, Diff: "diff2"},
		kube.ResourceKey{Kind: "Deployment", Namespace: "production", Name: "app2"}: {Type: Deleted, Diff: "diff3"},
		kube.ResourceKey{Kind: "ConfigMap", Namespace: "default", Name: "config"}:   {Type: Unchanged, Diff: ""},
	}

	// Filter resources in default namespace that have changes
	filtered := results.Apply(func(key kube.ResourceKey, result Result) bool {
		return key.Namespace == "default" && result.Type != Unchanged
	})

	assert.Equal(t, 2, len(filtered))
	assert.Contains(t, filtered, kube.ResourceKey{Kind: "Deployment", Namespace: "default", Name: "app1"})
	assert.Contains(t, filtered, kube.ResourceKey{Kind: "Service", Namespace: "default", Name: "app1"})
}

func TestResults_Analysis(t *testing.T) {
	results := Results{
		kube.ResourceKey{Kind: "Deployment", Name: "changed-app"}:   {Type: Changed, Diff: "changed diff"},
		kube.ResourceKey{Kind: "Service", Name: "created-service"}:  {Type: Created, Diff: "created diff"},
		kube.ResourceKey{Kind: "ConfigMap", Name: "deleted-config"}: {Type: Deleted, Diff: "deleted diff"},
		kube.ResourceKey{Kind: "Secret", Name: "unchanged-secret"}:  {Type: Unchanged, Diff: ""},
	}

	// Test HasChanges
	assert.True(t, results.HasChanges())

	noChangesResults := Results{
		kube.ResourceKey{Kind: "Secret", Name: "unchanged-secret"}: {Type: Unchanged, Diff: ""},
	}
	assert.False(t, noChangesResults.HasChanges())

	// Test IsEmpty
	assert.False(t, results.IsEmpty())
	emptyResults := Results{}
	assert.True(t, emptyResults.IsEmpty())

	// Test Count
	assert.Equal(t, 4, results.Count())
	assert.Equal(t, 0, emptyResults.Count())

	// Test CountByType
	assert.Equal(t, 1, results.CountByType(Changed))
	assert.Equal(t, 1, results.CountByType(Created))
	assert.Equal(t, 1, results.CountByType(Deleted))
	assert.Equal(t, 1, results.CountByType(Unchanged))
	assert.Equal(t, 0, results.CountByType(ChangeType(99))) // Invalid type

	// Test GetResourceKeys
	keys := results.GetResourceKeys()
	assert.Equal(t, 4, len(keys))

	// Test GetResourceKeysByType
	changedKeys := results.GetResourceKeysByType(Changed)
	assert.Equal(t, 1, len(changedKeys))
	assert.Equal(t, "changed-app", changedKeys[0].Name)

	createdKeys := results.GetResourceKeysByType(Created)
	assert.Equal(t, 1, len(createdKeys))
	assert.Equal(t, "created-service", createdKeys[0].Name)
}

func TestResults_GetStatistics(t *testing.T) {
	results := Results{
		kube.ResourceKey{Kind: "Deployment", Name: "app1"}:  {Type: Changed, Diff: "diff1"},
		kube.ResourceKey{Kind: "Deployment", Name: "app2"}:  {Type: Changed, Diff: "diff2"},
		kube.ResourceKey{Kind: "Service", Name: "svc1"}:     {Type: Created, Diff: "diff3"},
		kube.ResourceKey{Kind: "ConfigMap", Name: "config"}: {Type: Deleted, Diff: "diff4"},
		kube.ResourceKey{Kind: "Secret", Name: "secret1"}:   {Type: Unchanged, Diff: ""},
		kube.ResourceKey{Kind: "Secret", Name: "secret2"}:   {Type: Unchanged, Diff: ""},
	}

	stats := results.GetStatistics()

	assert.Equal(t, 6, stats.Total)
	assert.Equal(t, 2, stats.Changed)
	assert.Equal(t, 1, stats.Created)
	assert.Equal(t, 1, stats.Deleted)
	assert.Equal(t, 2, stats.Unchanged)

	// Test with empty results
	emptyResults := Results{}
	emptyStats := emptyResults.GetStatistics()
	assert.Equal(t, 0, emptyStats.Total)
	assert.Equal(t, 0, emptyStats.Changed)
	assert.Equal(t, 0, emptyStats.Created)
	assert.Equal(t, 0, emptyStats.Deleted)
	assert.Equal(t, 0, emptyStats.Unchanged)
}

func TestResults_StringSummary(t *testing.T) {
	results := Results{
		kube.ResourceKey{Kind: "Deployment", Namespace: "default", Name: "app1"}:    {Type: Changed, Diff: "diff1"},
		kube.ResourceKey{Kind: "Deployment", Namespace: "production", Name: "app2"}: {Type: Changed, Diff: "diff2"},
		kube.ResourceKey{Kind: "Service", Namespace: "default", Name: "svc1"}:       {Type: Created, Diff: "diff3"},
		kube.ResourceKey{Kind: "ConfigMap", Name: "config1"}:                        {Type: Deleted, Diff: "diff4"}, // cluster-scoped
		kube.ResourceKey{Kind: "Secret", Namespace: "default", Name: "secret1"}:     {Type: Unchanged, Diff: ""},
	}

	summary := results.StringSummary()

	// Check that each section is present
	assert.Contains(t, summary, "Unchanged:")
	assert.Contains(t, summary, "Changed:")
	assert.Contains(t, summary, "Create:")
	assert.Contains(t, summary, "Delete:")

	// Check specific resources are listed correctly
	assert.Contains(t, summary, "Secret/default/secret1") // Namespaced resource
	assert.Contains(t, summary, "Deployment/default/app1")
	assert.Contains(t, summary, "Deployment/production/app2")
	assert.Contains(t, summary, "Service/default/svc1")
	assert.Contains(t, summary, "ConfigMap/config1") // Cluster-scoped resource

	// Test with empty results
	emptyResults := Results{}
	emptySummary := emptyResults.StringSummary()
	assert.Equal(t, "", emptySummary)

	// Test with no changes (only unchanged)
	unchangedOnlyResults := Results{
		kube.ResourceKey{Kind: "Secret", Namespace: "default", Name: "secret1"}: {Type: Unchanged, Diff: ""},
	}
	unchangedSummary := unchangedOnlyResults.StringSummary()
	assert.Contains(t, unchangedSummary, "Unchanged:")
	assert.Contains(t, unchangedSummary, "Secret/default/secret1")
	assert.NotContains(t, unchangedSummary, "Changed:")
	assert.NotContains(t, unchangedSummary, "Create:")
	assert.NotContains(t, unchangedSummary, "Delete:")
}
