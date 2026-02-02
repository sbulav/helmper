package helm

import (
	"log/slog"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/jinzhu/copier"
	"helm.sh/helm/v3/pkg/cli"
)

func (collection ChartCollection) pull(settings *cli.EnvSettings) error {
	for _, chart := range collection.Charts {
		if _, err := chart.Pull(settings); err != nil {
			return err
		}
	}
	return nil
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
		Verbose: false,
		Update:  false,
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
	for _, c := range collection.Charts {
		vs, err := c.ResolveVersions(settings)
		if err != nil {
			// resolve Glob version
			v, err := c.ResolveVersion(settings)
			if err != nil {
				slog.Error("failed to resolve chart version",
					slog.String("name", c.Name),
					slog.String("version", c.Version),
					slog.String("repo", c.Repo.URL),
					slog.Any("error", err))
				continue
			}
			c.Version = v
			res = append(res, c)
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
	collection.Charts = res

	// Pull Helm Charts
	err = collection.pull(settings)
	if err != nil {
		return nil, err
	}
	slog.Info("Pulled Helm Charts")

	return to.Ptr(collection), nil
}
