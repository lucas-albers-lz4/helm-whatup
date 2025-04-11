// Package main provides a Helm plugin to check for outdated chart releases.
// It compares installed chart versions with the latest available versions in repositories.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
)

// Output format options
const (
	outputFormatPlain = "plain"
	outputFormatShort = "short"
	outputFormatJSON  = "json"
	outputFormatYAML  = "yaml"
	outputFormatYML   = "yml"
	outputFormatTable = "table"
)

// Status constants for chart versions
const (
	statusOutdated = "OUTDATED"
	statusUptodate = "UPTODATE"
)

var (
	outputFormat string
	devel        bool
	tlsEnable    bool
	tlsHostname  string
	tlsCaCert    string
	tlsCert      string
	tlsKey       string
	tlsVerify    bool
)

var version = "canary"

// ChartVersionInfo stores information about a chart's version status
// including the installed version and the latest available version.
type ChartVersionInfo struct {
	ReleaseName      string `json:"releaseName"`
	ChartName        string `json:"chartName"`
	InstalledVersion string `json:"installedVersion"`
	LatestVersion    string `json:"latestVersion"`
	Status           string `json:"status"`
}

func main() {
	cmd := &cobra.Command{
		Use:   "whatup [flags]",
		Short: fmt.Sprintf("check if installed charts are out of date (helm-whatup %s)", version),
		RunE:  run,
	}

	f := cmd.Flags()

	f.StringVarP(&outputFormat, "output", "o", outputFormatPlain, "output format. Accepted formats: plain, json, yaml, table, short")
	f.BoolVarP(&devel, "devel", "d", false, "whether to include pre-releases or not")
	f.BoolVar(&tlsEnable, "tls", false, "enable TLS for requests to the server")
	f.StringVar(&tlsCaCert, "tls-ca-cert", "", "path to TLS CA certificate file")
	f.StringVar(&tlsCert, "tls-cert", "", "path to TLS certificate file")
	f.StringVar(&tlsKey, "tls-key", "", "path to TLS key file")
	f.StringVar(&tlsHostname, "tls-hostname", "", "the server name used to verify the hostname on the returned certificates from the server")
	f.BoolVar(&tlsVerify, "tls-verify", false, "enable TLS for requests to the server, and controls whether the client verifies the server's certificate chain and host name")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newClient() (*action.Configuration, error) {
	settings := cli.New()
	actionConfig := new(action.Configuration)

	// Initialize action configuration
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), debug); err != nil {
		return nil, fmt.Errorf("failed to initialize Helm client: %w", err)
	}

	return actionConfig, nil
}

func debug(format string, v ...interface{}) {
	// Suppress debug output by default
	if os.Getenv("HELM_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, format, v...)
	}
}

func run(_ *cobra.Command, _ []string) error {
	actionConfig, err := newClient()
	if err != nil {
		return err
	}

	releases, err := fetchReleases(actionConfig)
	if err != nil {
		return err
	}

	repositories, err := fetchIndices()
	if err != nil {
		return err
	}

	if len(releases) == 0 {
		if outputFormat == outputFormatPlain {
			fmt.Println("No releases found. All up to date!")
		}
		return nil
	}

	if len(repositories) == 0 {
		if outputFormat == outputFormatPlain {
			fmt.Println("No repositories found. Did you run `helm repo update`?")
		}
		return nil
	}

	var result []ChartVersionInfo

	for _, release := range releases {
		chartName := release.Chart.Metadata.Name
		chartVersion := release.Chart.Metadata.Version

		// Check each repository for this chart
		for _, idx := range repositories {
			// Check if the chart exists in this repository
			entries, exists := idx.Entries[chartName]
			if !exists || len(entries) == 0 {
				continue
			}

			// Find the latest version
			var latestVersion string

			// Sort versions (already sorted in the index)
			// Get the latest version
			for _, entry := range entries {
				// If devel is false and this is a prerelease, skip it
				if !devel && entry.APIVersion == "prerelease" {
					continue
				}
				latestVersion = entry.Version
				break
			}

			if latestVersion == "" {
				continue
			}

			versionStatus := ChartVersionInfo{
				ReleaseName:      release.Name,
				ChartName:        chartName,
				InstalledVersion: chartVersion,
				LatestVersion:    latestVersion,
			}

			if versionStatus.InstalledVersion == versionStatus.LatestVersion {
				versionStatus.Status = statusUptodate
			} else {
				versionStatus.Status = statusOutdated
			}
			result = append(result, versionStatus)

			// Found a match for this chart, no need to check other repositories
			break
		}
	}

	switch outputFormat {
	case outputFormatPlain:
		for _, versionInfo := range result {
			if versionInfo.LatestVersion != versionInfo.InstalledVersion {
				fmt.Printf("There is an update available for release %s (%s)!\n"+
					"Installed version: %s\n"+
					"Available version: %s\n",
					versionInfo.ReleaseName,
					versionInfo.ChartName,
					versionInfo.InstalledVersion,
					versionInfo.LatestVersion)
			} else {
				fmt.Printf("Release %s (%s) is up to date.\n", versionInfo.ReleaseName, versionInfo.LatestVersion)
			}
		}
		fmt.Println("Done.")
	case outputFormatShort:
		for _, versionInfo := range result {
			if versionInfo.LatestVersion != versionInfo.InstalledVersion {
				fmt.Printf("%s (%s): %s --> %s\n", versionInfo.ReleaseName, versionInfo.ChartName, versionInfo.InstalledVersion, versionInfo.LatestVersion)
			}
		}
	case outputFormatJSON:
		outputBytes, err := json.MarshalIndent(result, "", "    ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(outputBytes))
	case outputFormatYML, outputFormatYAML:
		outputBytes, err := yaml.Marshal(result)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
		fmt.Println(string(outputBytes))
	case outputFormatTable:
		table := uitable.New()
		table.AddRow("RELEASE", "CHART", "INSTALLED_VERSION", "LATEST_VERSION", "STATUS")
		for _, versionInfo := range result {
			table.AddRow(versionInfo.ReleaseName, versionInfo.ChartName, versionInfo.InstalledVersion, versionInfo.LatestVersion, versionInfo.Status)
		}
		fmt.Println(table)
	default:
		return fmt.Errorf("invalid formatter: %s", outputFormat)
	}

	return nil
}

func fetchReleases(actionConfig *action.Configuration) ([]*release.Release, error) {
	listAction := action.NewList(actionConfig)
	// Configure the list action
	listAction.All = true
	listAction.AllNamespaces = false

	releases, err := listAction.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to list releases: %w", err)
	}
	return releases, nil
}

func fetchIndices() ([]*repo.IndexFile, error) {
	indices := []*repo.IndexFile{}
	settings := cli.New()

	// Get repositories file
	repoFile := settings.RepositoryConfig

	// Load repositories
	repoFileData, err := repo.LoadFile(repoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load repository file: %w", err)
	}

	for _, repoEntry := range repoFileData.Repositories {
		// Construct the index file path
		indexFileName := repoEntry.Name + "-index.yaml"
		cachePath := filepath.Join(settings.RepositoryCache, indexFileName)

		// Load the index file
		indexFile, err := repo.LoadIndexFile(cachePath)
		if err != nil {
			// Skip repositories with errors
			continue
		}

		indices = append(indices, indexFile)
	}

	return indices, nil
}
