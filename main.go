// Package main provides a Helm plugin to check for outdated chart releases.
// It compares installed chart versions with the latest available versions in repositories.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	Namespace        string `json:"namespace"`
	ChartName        string `json:"chartName"`
	InstalledVersion string `json:"installedVersion"`
	LatestVersion    string `json:"latestVersion"`
	RepoName         string `json:"repoName"`
	Status           string `json:"status"`
}

func main() {
	cmd := &cobra.Command{
		Use:   "whatup [flags]",
		Short: fmt.Sprintf("check if installed charts are out of date (helm-whatup %s)", version),
		RunE:  run,
	}

	f := cmd.Flags()

	f.StringVarP(&outputFormat, "output", "o", outputFormatTable, "output format. Accepted formats: plain, json, yaml, table, short")
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

	// Use "" for namespace to get all namespaces
	if err := actionConfig.Init(settings.RESTClientGetter(), "", os.Getenv("HELM_DRIVER"), debug); err != nil {
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

	// Get repository file data for reference
	settings := cli.New()
	repoFile := settings.RepositoryConfig
	repoFileData, err := repo.LoadFile(repoFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: Failed to load repository file: %v\n", err)
	}

	// Create a map of chart names to repositories for quick lookup
	chartRepoMap := make(map[string]string)
	for _, repo := range repoFileData.Repositories {
		// Get the index file for this repository
		for _, idx := range repositories {
			// Check each chart in this repository
			for chartName := range idx.Entries {
				// Associate this chart with this repository
				// Only if not already set or if the repository name matches the chart name partly
				if _, exists := chartRepoMap[chartName]; !exists || strings.Contains(repo.Name, chartName) {
					chartRepoMap[chartName] = repo.Name
				}
			}
		}
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

	// Create a warning message buffer
	var warnings []string

	var result []ChartVersionInfo

	for _, release := range releases {
		chartName := release.Chart.Metadata.Name
		chartVersion := release.Chart.Metadata.Version
		repoName := ""
		chartFound := false

		// Try to find the repository from annotations or labels
		if release.Chart.Metadata.Annotations != nil {
			if val, ok := release.Chart.Metadata.Annotations["artifacthub.io/repository"]; ok {
				repoName = val
			}
		}

		// If we haven't found a repo name, check our map
		if repoName == "" {
			if repo, exists := chartRepoMap[chartName]; exists {
				repoName = repo
			}
		}

		// For each chart, check all repositories
		for _, idx := range repositories {
			// Check if the chart exists in this repository
			entries, exists := idx.Entries[chartName]
			if !exists || len(entries) == 0 {
				continue
			}

			chartFound = true

			// Find the latest version
			latestVersion := ""

			// Get the latest version (index is already sorted with latest first)
			for _, entry := range entries {
				// Skip prerelease versions if devel flag is not set
				if !devel && entry.APIVersion == "prerelease" {
					continue
				}
				latestVersion = entry.Version

				// If repository name is not set, try to find it
				if repoName == "" && len(entry.URLs) > 0 {
					// Extract repository URL from chart URL
					chartURL := entry.URLs[0]

					// Try to match with known repositories
					for _, repo := range repoFileData.Repositories {
						if strings.Contains(chartURL, repo.URL) {
							repoName = repo.Name
							break
						}
					}
				}

				break
			}

			if latestVersion == "" {
				continue
			}

			// Try different methods to find the repository name
			if repoName == "" {
				// Method 1: Check if this chart name is a known repo
				for _, repo := range repoFileData.Repositories {
					if repo.Name == chartName {
						repoName = repo.Name
						break
					}
				}

				// Method 2: Try to parse from index file name
				if repoName == "" && idx.APIVersion != "" {
					// Look for repo name in the index metadata
					// The APIVersion field is sometimes used to store the path
					nameFromPath := filepath.Base(idx.APIVersion)
					if strings.HasSuffix(nameFromPath, "-index.yaml") {
						repoName = strings.TrimSuffix(nameFromPath, "-index.yaml")
					}
				}

				// Method 3: Match by URL patterns
				if repoName == "" && len(entries) > 0 && len(entries[0].URLs) > 0 {
					chartURL := entries[0].URLs[0]

					// Common URL patterns
					for _, repo := range repoFileData.Repositories {
						if strings.Contains(chartURL, repo.URL) {
							repoName = repo.Name
							break
						}
					}

					// Try to extract repo name from URL
					if repoName == "" {
						// Example: https://charts.bitnami.com/bitnami
						// Extract "bitnami"
						parts := strings.Split(chartURL, "/")
						if len(parts) >= 3 {
							domainParts := strings.Split(parts[2], ".")
							if len(domainParts) >= 2 {
								repoName = domainParts[1]
							}
						}
					}
				}

				// Last resort: Check the release name
				if repoName == "" {
					// For example, rke2-cilium likely comes from rke2-charts
					for _, repo := range repoFileData.Repositories {
						prefix := strings.Split(chartName, "-")[0]
						if strings.HasPrefix(repo.Name, prefix) {
							repoName = repo.Name
							break
						}
					}
				}

				// Final fallback
				if repoName == "" {
					repoName = "unknown"
				}
			}

			versionStatus := ChartVersionInfo{
				ReleaseName:      release.Name,
				Namespace:        release.Namespace,
				ChartName:        chartName,
				InstalledVersion: chartVersion,
				LatestVersion:    latestVersion,
				RepoName:         repoName,
			}

			// Simple string comparison may not work correctly for semver
			// Using equal instead of direct string comparison
			if versionStatus.InstalledVersion == versionStatus.LatestVersion {
				versionStatus.Status = statusUptodate
			} else {
				versionStatus.Status = statusOutdated
			}

			result = append(result, versionStatus)

			// Found a match for this chart, no need to check other repositories
			break
		}

		// Output warning if chart's repo couldn't be determined
		if !chartFound {
			warnings = append(warnings, fmt.Sprintf("The source repository could not be determined for '%s'", release.Name))
		}
	}

	// Print collected warnings if in plain format
	if outputFormat == outputFormatPlain && len(warnings) > 0 {
		fmt.Println()
		for _, warning := range warnings {
			fmt.Printf("WARNING: %s\n", warning)
		}
	}

	return formatAndPrintResults(result)
}

// formatAndPrintResults formats and prints the version information based on the selected output format
func formatAndPrintResults(result []ChartVersionInfo) error {
	// Check if we have any outdated charts
	hasOutdated := false
	for _, versionInfo := range result {
		if versionInfo.Status == statusOutdated {
			hasOutdated = true
			break
		}
	}

	// If no outdated charts and plain format, show a simpler message
	if !hasOutdated && outputFormat == outputFormatPlain {
		fmt.Println("No charts need updates. All up to date!")
		return nil
	}

	switch outputFormat {
	case outputFormatPlain:
		fmt.Println("\nWARNING: Charts marked as deprecated will not be shown in the results.\n")
		for _, versionInfo := range result {
			if versionInfo.LatestVersion != versionInfo.InstalledVersion {
				fmt.Printf("There is an update available for release %s (%s)!\n"+
					"Installed version: %s\n"+
					"Available version: %s\n\n",
					versionInfo.ReleaseName,
					versionInfo.ChartName,
					versionInfo.InstalledVersion,
					versionInfo.LatestVersion)
			} else {
				fmt.Printf("Release %s (%s) is up to date.\n", versionInfo.ReleaseName, versionInfo.ChartName)
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
		fmt.Println("\nWARNING: Charts marked as deprecated will not be shown in the results.\n")

		// Show outdated charts
		table := uitable.New()
		table.MaxColWidth = 50
		table.Wrap = true

		// Add column padding
		table.Separator = "  "

		table.AddRow("NAME", "NAMESPACE", "INSTALLED VERSION", "LATEST VERSION", "CHART", "REPOSITORY")

		for _, versionInfo := range result {
			if versionInfo.LatestVersion != versionInfo.InstalledVersion {
				// Use the correct namespace from the release
				table.AddRow(
					versionInfo.ReleaseName,
					versionInfo.Namespace,
					versionInfo.InstalledVersion,
					versionInfo.LatestVersion,
					versionInfo.ChartName,
					versionInfo.RepoName,
				)
			}
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
	listAction.AllNamespaces = true // Make sure we get releases from all namespaces
	listAction.SetStateMask()       // Make sure we get all release states

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
