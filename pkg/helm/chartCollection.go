package helm

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/jinzhu/copier"
	"helm.sh/helm/v3/pkg/cli"
)

func (collection ChartCollection) pull(settings *cli.EnvSettings, continueOnError bool) (*ChartCollection, error) {
	res := []*Chart{}
	pullErrs := []error{}
	for _, chart := range collection.Charts {
		started := time.Now()
		slog.Info("Pulling Helm chart",
			slog.String("chart", chart.Name),
			slog.String("version", chart.Version),
			slog.String("repo", chart.Repo.URL))
		if _, err := chart.Pull(settings); err != nil {
			pullErr := fmt.Errorf("pull chart %s@%s: %w", chart.Name, chart.Version, err)
			if !continueOnError {
				return nil, pullErr
			}

			pullErrs = append(pullErrs, pullErr)
			slog.Warn("skipping chart after pull failure",
				slog.String("chart", chart.Name),
				slog.String("version", chart.Version),
				slog.String("repo", chart.Repo.URL),
				slog.Any("error", err))
			continue
		}
		res = append(res, chart)
		slog.Info("Pulled Helm chart",
			slog.String("chart", chart.Name),
			slog.String("version", chart.Version),
			slog.Duration("duration", time.Since(started)))
	}
	if len(pullErrs) > 0 {
		slog.Warn("some Helm charts failed to pull and were skipped", slog.Int("count", len(pullErrs)), slog.Any("error", errors.Join(pullErrs...)))
	}
	collection.Charts = res
	return &collection, nil
}

func (collection ChartCollection) addToHelmRepositoryConfig(settings *cli.EnvSettings) error {
	for _, c := range collection.Charts {
		if strings.HasPrefix(c.Repo.URL, "oci://") {
			continue
		}
		_, err := c.addToHelmRepositoryFile(settings)
		if err != nil {
			return err
		}

	}
	return nil
}

// configures helm and pulls charts to local fs
func (collection ChartCollection) SetupHelm(settings *cli.EnvSettings, setters ...Option) (*ChartCollection, error) {

	// Default Options
	args := &Options{
		Verbose:               false,
		Update:                false,
		ContinueOnChartErrors: false,
	}

	for _, setter := range setters {
		setter(args)
	}

	// Add Helm Repos
	err := collection.addToHelmRepositoryConfig(settings)
	if err != nil {
		return nil, err
	}
	slog.Debug("Added Helm repositories to config", slog.String("config_path", settings.RepositoryConfig))

	// Update Helm Repos
	output, err := updateRepositories(settings, args.Verbose, args.Update)
	if err != nil {
		return nil, err
	}
	// Log results
	if args.Verbose {
		slog.Debug("Updated all Helm repositories", slog.String("output", output))
	} else {
		slog.Info("Updated all Helm repositories")
	}

	// Expand collection if semantic version range
	res := []*Chart{}
	resolveErrs := []error{}
	for _, c := range collection.Charts {
		vs, rangeErr := c.ResolveVersions(settings)
		if rangeErr != nil {
			// resolve Glob version
			v, err := c.ResolveVersion(settings)
			if err != nil {
				resolveErr := fmt.Errorf("resolve chart %s@%s from %s: range resolution failed: %w; fallback resolution failed: %w", c.Name, c.Version, c.Repo.URL, rangeErr, err)
				resolveErrs = append(resolveErrs, resolveErr)
				slog.Error("failed to resolve chart version",
					slog.String("name", c.Name),
					slog.String("version", c.Version),
					slog.String("repo", c.Repo.URL),
					slog.Any("range_error", rangeErr),
					slog.Any("error", err))
				continue
			}
			c.Version = v
			res = append(res, c)
			continue
		}

		if len(vs) == 0 {
			resolveErr := fmt.Errorf("resolve chart %s@%s from %s: no matching versions found", c.Name, c.Version, c.Repo.URL)
			resolveErrs = append(resolveErrs, resolveErr)
			slog.Error("failed to resolve chart version",
				slog.String("name", c.Name),
				slog.String("version", c.Version),
				slog.String("repo", c.Repo.URL),
				slog.Any("error", resolveErr))
			continue
		}

		// If LatestVersionOnly is enabled and we have multiple versions,
		// only keep the latest (last) one since versions are sorted
		if args.LatestVersionOnly && len(vs) > 1 {
			vs = vs[len(vs)-1:]
		}

		for _, v := range vs {
			cv := &Chart{}
			err := copier.Copy(&cv, &c)
			if err != nil {
				return nil, err
			}
			cv.Version = v
			res = append(res, cv)
		}
	}
	if len(resolveErrs) > 0 && !args.ContinueOnChartErrors {
		return nil, errors.Join(resolveErrs...)
	}
	if len(resolveErrs) > 0 {
		slog.Warn("some Helm charts failed to resolve and were skipped", slog.Int("count", len(resolveErrs)), slog.Any("error", errors.Join(resolveErrs...)))
	}
	collection.Charts = res

	// Pull Helm Charts
	pulledCollection, err := collection.pull(settings, args.ContinueOnChartErrors)
	if err != nil {
		return nil, err
	}
	collection = *pulledCollection
	slog.Info("Pulled Helm Charts")

	return to.Ptr(collection), nil
}
