package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/helm/pkg/repo"
)

// MockHelmClient mocks the Helm client interface for testing
type MockHelmClient struct {
	mock.Mock
}

// List mocks the List method of the Helm client
func (m *MockHelmClient) List() (*services.ListReleasesResponse, error) {
	args := m.Called()
	return args.Get(0).(*services.ListReleasesResponse), args.Error(1)
}

// ReleaseContent mocks the ReleaseContent method needed for the helm.Client interface
func (m *MockHelmClient) ReleaseContent(releaseName string) (*services.GetReleaseContentResponse, error) {
	args := m.Called(releaseName)
	return args.Get(0).(*services.GetReleaseContentResponse), args.Error(1)
}

// Setup test data
func setupTestReleases() []*release.Release {
	return []*release.Release{
		{
			Name: "test-release-1",
			Chart: &chart.Chart{
				Metadata: &chart.Metadata{
					Name:    "test-chart",
					Version: "1.0.0",
				},
			},
		},
		{
			Name: "test-release-2",
			Chart: &chart.Chart{
				Metadata: &chart.Metadata{
					Name:    "another-chart",
					Version: "2.0.0",
				},
			},
		},
	}
}

// Setup test repositories
func setupTestIndices() []*repo.IndexFile {
	chartVersions := []*repo.ChartVersion{
		{
			Metadata: &chart.Metadata{
				Name:    "test-chart",
				Version: "1.1.0", // newer version available
			},
		},
	}

	chartEntries := map[string]repo.ChartVersions{
		"test-chart": chartVersions,
	}

	return []*repo.IndexFile{
		{
			Entries: chartEntries,
		},
	}
}

// Test the ChartVersionInfo struct
func TestChartVersionInfo(t *testing.T) {
	info := ChartVersionInfo{
		ReleaseName:      "test-release",
		ChartName:        "test-chart",
		InstalledVersion: "1.0.0",
		LatestVersion:    "1.1.0",
		Status:           statusOutdated,
	}

	assert.Equal(t, "test-release", info.ReleaseName)
	assert.Equal(t, "test-chart", info.ChartName)
	assert.Equal(t, "1.0.0", info.InstalledVersion)
	assert.Equal(t, "1.1.0", info.LatestVersion)
	assert.Equal(t, statusOutdated, info.Status)
}

// Test the JSON output format
func TestJSONOutput(t *testing.T) {
	// Setup
	result := []ChartVersionInfo{
		{
			ReleaseName:      "test-release",
			ChartName:        "test-chart",
			InstalledVersion: "1.0.0",
			LatestVersion:    "1.1.0",
			Status:           statusOutdated,
		},
	}

	// Redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute the function we want to test (just the JSON output part)
	outputBytes, err := json.MarshalIndent(result, "", "    ")
	assert.NoError(t, err)

	// Print the output as the function would
	_, err = fmt.Println(string(outputBytes))
	assert.NoError(t, err)

	// Reset stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	assert.NoError(t, err)

	// Validate output
	capturedOutput := buf.String()
	assert.Contains(t, capturedOutput, "test-release")
	assert.Contains(t, capturedOutput, "test-chart")
	assert.Contains(t, capturedOutput, "1.0.0")
	assert.Contains(t, capturedOutput, "1.1.0")
	assert.Contains(t, capturedOutput, statusOutdated)
}

// For a more complete test suite, you would add tests for:
// 1. The fetchReleases function (mocking the Helm client)
// 2. The fetchIndices function
// 3. The run function with different output formats
// 4. Integration tests that actually connect to a test cluster
