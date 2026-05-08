package helm

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blang/semver/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
)

type MockIndexFileLoader struct {
	mock.Mock
}

func (m *MockIndexFileLoader) LoadIndexFile(indexFilePath string) (*repo.IndexFile, error) {
	args := m.Called(indexFilePath)
	return args.Get(0).(*repo.IndexFile), args.Error(1)
}

func TestVersionsInRange(t *testing.T) {
	mockRepoIndex := &repo.IndexFile{
		Entries: map[string]repo.ChartVersions{
			"testchart": {
				{
					Metadata: &chart.Metadata{
						Name:    "testchart",
						Version: "1.0.0",
					},
				},
				{
					Metadata: &chart.Metadata{
						Name:    "testchart",
						Version: "1.1.0",
					},
				},
			},
		},
	}

	mockLoader := new(MockIndexFileLoader)
	mockLoader.On("LoadIndexFile", mock.Anything).Return(mockRepoIndex, nil)

	r, _ := semver.ParseRange(">= 1.0.0")
	c := Chart{
		Repo:            repo.Entry{Name: "testrepo"},
		Name:            "testchart",
		Version:         "1.0.0",
		IndexFileLoader: mockLoader,
	}
	versions, err := VersionsInRange(r, c)
	assert.NoError(t, err)
	assert.Equal(t, []string{"1.0.0", "1.1.0"}, versions)

	mockLoader.AssertExpectations(t)
}

func TestResolveVersions(t *testing.T) {
	mockClient := new(MockRegistryClient)

	c := Chart{
		Repo: repo.Entry{
			URL: "oci://localhost:5000/testchart",
		},
		Name:           "testchart",
		Version:        ">= 1.0.0",
		PlainHTTP:      true,
		RegistryClient: mockClient,
	}

	settings := cli.New()

	mockClient.On("Tags", mock.Anything).Return([]string{"1.0.0", "1.1.0"}, nil)

	versions, err := c.ResolveVersions(settings)
	assert.NoError(t, err)
	assert.Equal(t, []string{"1.0.0", "1.1.0"}, versions)
}

func TestResolveVersion(t *testing.T) {
	mockClient := new(MockRegistryClient)

	settings := cli.New()

	c := Chart{
		Repo: repo.Entry{
			URL: "oci://localhost:5000/testchart",
		},
		Name:           "testchart",
		Version:        ">= 1.0.0",
		PlainHTTP:      true,
		RegistryClient: mockClient,
	}

	mockClient.On("Tags", mock.Anything).Return([]string{"1.0.0", "1.1.0"}, nil)

	version, err := c.ResolveVersion(settings)
	assert.NoError(t, err)
	assert.Equal(t, "1.1.0", version)
}

func TestResolveVersionWithVPrefixedRange(t *testing.T) {
	mockClient := new(MockRegistryClient)

	settings := cli.New()

	c := Chart{
		Repo: repo.Entry{
			URL: "oci://localhost:5000/testchart",
		},
		Name:           "testchart",
		Version:        ">=v1.0.0",
		PlainHTTP:      true,
		RegistryClient: mockClient,
	}

	mockClient.On("Tags", mock.Anything).Return([]string{"1.0.0", "1.1.0"}, nil)

	version, err := c.ResolveVersion(settings)
	assert.NoError(t, err)
	assert.Equal(t, "v1.1.0", version)
}

func TestLatestVersionSkipsPrereleases(t *testing.T) {
	mockClient := new(MockRegistryClient)

	c := Chart{
		Repo: repo.Entry{
			URL: "oci://localhost:5000/testchart",
		},
		Name:           "testchart",
		Version:        ">= 1.0.0",
		PlainHTTP:      true,
		RegistryClient: mockClient,
	}

	settings := cli.New()

	mockClient.On("Tags", mock.Anything).Return([]string{"1.0.0", "1.1.0", "1.2.0-rc.1"}, nil)

	version, err := c.LatestVersion(settings)
	assert.NoError(t, err)
	assert.Equal(t, "1.1.0", version)
}

func TestChartOptionWithoutImageDetectionSkipsParsing(t *testing.T) {
	chart := &Chart{
		Repo: repo.Entry{
			URL: "invalid://unused",
		},
		Name:    "testchart",
		Version: "1.0.0",
	}
	option := ChartOption{
		ChartCollection: &ChartCollection{Charts: []*Chart{chart}},
		IdentifyImages:  false,
	}

	data, err := option.Run(context.Background())
	assert.NoError(t, err)
	assert.Contains(t, data, chart)
	assert.Len(t, data[chart], 0)
}

func TestSetupHelmFailsWhenChartResolutionFails(t *testing.T) {
	mockClient := new(MockRegistryClient)
	resolveErr := errors.New("registry unavailable")

	settings := cli.New()
	settings.RepositoryCache = t.TempDir()
	settings.RepositoryConfig = filepath.Join(settings.RepositoryCache, "repositories.yaml")
	assert.NoError(t, repo.NewFile().WriteFile(settings.RepositoryConfig, 0o644))

	collection := ChartCollection{Charts: []*Chart{
		{
			Repo: repo.Entry{
				URL: "oci://localhost:5000/testchart",
			},
			Name:           "testchart",
			Version:        ">=1.0.0",
			PlainHTTP:      true,
			RegistryClient: mockClient,
		},
	}}

	mockClient.On("Tags", mock.Anything).Return([]string{}, resolveErr).Twice()

	_, err := collection.SetupHelm(settings)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "range resolution failed"), err.Error())
	assert.True(t, strings.Contains(err.Error(), "fallback resolution failed"), err.Error())

	mockClient.AssertExpectations(t)
}

func TestSetupHelmContinuesWhenChartResolutionFails(t *testing.T) {
	mockClient := new(MockRegistryClient)
	resolveErr := errors.New("registry unavailable")

	settings := cli.New()
	settings.RepositoryCache = t.TempDir()
	settings.RepositoryConfig = filepath.Join(settings.RepositoryCache, "repositories.yaml")
	assert.NoError(t, repo.NewFile().WriteFile(settings.RepositoryConfig, 0o644))

	collection := ChartCollection{Charts: []*Chart{
		{
			Repo: repo.Entry{
				URL: "oci://localhost:5000/testchart",
			},
			Name:           "testchart",
			Version:        ">=1.0.0",
			PlainHTTP:      true,
			RegistryClient: mockClient,
		},
	}}

	mockClient.On("Tags", mock.Anything).Return([]string{}, resolveErr).Twice()

	charts, err := collection.SetupHelm(settings, ContinueOnChartErrors(true))
	assert.NoError(t, err)
	assert.Len(t, charts.Charts, 0)

	mockClient.AssertExpectations(t)
}

func TestLatestVersion(t *testing.T) {
	mockClient := new(MockRegistryClient)

	c := Chart{
		Repo: repo.Entry{
			URL: "oci://localhost:5000/testchart",
		},
		Name:           "testchart",
		Version:        ">= 1.0.0",
		PlainHTTP:      true,
		RegistryClient: mockClient,
	}

	settings := cli.New()

	mockClient.On("Tags", mock.Anything).Return([]string{"1.0.0", "1.1.0"}, nil)

	version, err := c.LatestVersion(settings)
	assert.NoError(t, err)
	assert.Equal(t, "1.1.0", version)
}
