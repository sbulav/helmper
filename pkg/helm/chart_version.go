package helm

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/blang/semver/v4"
	"golang.org/x/xerrors"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
)

func VersionsInRange(r semver.Range, c Chart) ([]string, error) {
	prefixV := strings.Contains(c.Version, "v")
	config := cli.New()
	indexPath := fmt.Sprintf("%s/%s-index.yaml", config.RepositoryCache, c.Repo.Name)
	index, err := c.IndexFileLoader.LoadIndexFile(indexPath)
	if err != nil {
		return nil, err
	}
	return versionsInRangeFromIndex(r, index.Entries[c.Name], prefixV), nil
}

func versionsInRangeFromIndex(r semver.Range, versions repo.ChartVersions, prefixV bool) []string {
	vs := []semver.Version{}
	for _, v := range versions {
		sv, err := semver.ParseTolerant(v.Version)
		if err != nil {
			continue
		}
		if len(sv.Pre) > 0 {
			continue
		}
		if r(sv) {
			vs = append(vs, sv)
		}
	}

	semver.Sort(vs)

	versionsInRange := []string{}
	for _, v := range vs {
		s := v.String()
		if prefixV {
			s = "v" + s
		}
		versionsInRange = append(versionsInRange, s)
	}
	return versionsInRange
}

func latestStableVersion(versions []semver.Version, prefixV bool) (string, error) {
	stableVersions := []semver.Version{}
	for _, v := range versions {
		if len(v.Pre) == 0 {
			stableVersions = append(stableVersions, v)
		}
	}
	if len(stableVersions) == 0 {
		return "", xerrors.New("Not Found")
	}

	semver.Sort(stableVersions)
	latest := stableVersions[len(stableVersions)-1].String()
	if prefixV {
		return "v" + latest, nil
	}
	return latest, nil
}

func normalizeVersionRange(version string) string {
	return strings.ReplaceAll(strings.ReplaceAll(version, "*", "x"), "v", "")
}

func (c Chart) ResolveVersions(settings *cli.EnvSettings) ([]string, error) {
	version := normalizeVersionRange(c.Version)
	r, err := semver.ParseRange(version)
	if err != nil {
		return nil, err
	}

	slog.Debug("resolving chart versions",
		slog.String("chart", c.Name),
		slog.String("repo", c.Repo.Name),
		slog.String("version_range", c.Version))

	if strings.HasPrefix(c.Repo.URL, "oci://") {
		url, _ := strings.CutPrefix(c.Repo.URL, "oci://")
		// Append chart name to URL if not already present
		if !strings.HasSuffix(url, c.Name) {
			url = url + "/" + c.Name
		}
		tags, err := c.RegistryClient.Tags(url)
		if err != nil {
			return nil, err
		}

		vs := []semver.Version{}
		for _, t := range tags {
			s, err := semver.ParseTolerant(t)
			if err != nil {
				// non semver tag
				continue
			}
			vs = append(vs, s)
		}

		semver.Sort(vs)

		prefixV := strings.Contains(c.Version, "v")
		versionsInRange := []string{}
		for _, v := range vs {
			if len(v.Pre) > 0 {
				continue
			}
			if r(v) {
				s := v.String()
				if prefixV {
					s = "v" + s
				}
				versionsInRange = append(versionsInRange, s)
			}
		}
		return versionsInRange, nil
	}

	update, err := c.addToHelmRepositoryFile(settings)
	if err != nil {
		return nil, err
	}
	if update {
		_, err = updateRepositories(settings, false, false)
		if err != nil {
			return nil, err
		}
	}
	indexPath := fmt.Sprintf("%s/%s-index.yaml", settings.RepositoryCache, c.Repo.Name)
	index, err := c.IndexFileLoader.LoadIndexFile(indexPath)
	if err != nil {
		return nil, err
	}
	return versionsInRangeFromIndex(r, index.Entries[c.Name], strings.Contains(c.Version, "v")), nil
}

func (c Chart) ResolveVersion(settings *cli.EnvSettings) (string, error) {
	v := normalizeVersionRange(c.Version)
	r, err := semver.ParseRange(v)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(c.Repo.URL, "oci://") {
		url, _ := strings.CutPrefix(c.Repo.URL, "oci://")
		// Append chart name to URL if not already present
		if !strings.HasSuffix(url, c.Name) {
			url = url + "/" + c.Name
		}
		tags, err := c.RegistryClient.Tags(url)
		if err != nil {
			return "", err
		}

		vs := []semver.Version{}
		for _, t := range tags {
			s, err := semver.ParseTolerant(t)
			if err != nil {
				// non semver tag
				continue
			}
			vs = append(vs, s)
		}

		semver.Sort(vs)

		vs2 := []string{}
		for _, v := range vs {
			if len(v.Pre) > 0 {
				continue
			}
			if r(v) {
				vs2 = append(vs2, v.String())
			}
		}

		if len(vs2) == 0 {
			return "", fmt.Errorf("failed to resolve version for %s range; available tags: %+v", c.Version, tags)
		}

		prefixV := strings.Contains(c.Version, "v")
		if prefixV {
			return "v" + vs2[len(vs2)-1], nil
		}

		return vs2[len(vs2)-1], nil
	}

	update, err := c.addToHelmRepositoryFile(settings)
	if err != nil {
		return "", err
	}
	if update {
		_, err = updateRepositories(settings, false, false)
		if err != nil {
			return "", err
		}
	}

	indexPath := fmt.Sprintf("%s/%s-index.yaml", settings.RepositoryCache, c.Repo.Name)
	index, err := repo.LoadIndexFile(indexPath)
	if err != nil {
		return "", err
	}
	versions := versionsInRangeFromIndex(r, index.Entries[c.Name], strings.Contains(c.Version, "v"))
	if len(versions) > 0 {
		v := versions[len(versions)-1]
		slog.Debug("Resolved chart version", slog.String("chart", c.Name), slog.String("version", v))
		return v, nil
	}
	return "", xerrors.New("Not Found")
}

func chartVersionsToSemver(versions repo.ChartVersions) []semver.Version {
	vs := []semver.Version{}
	for _, v := range versions {
		sv, err := semver.ParseTolerant(v.Version)
		if err != nil {
			continue
		}
		vs = append(vs, sv)
	}
	return vs
}

func (c Chart) LatestVersion(settings *cli.EnvSettings) (string, error) {

	if strings.HasPrefix(c.Repo.URL, "oci://") {
		url, _ := strings.CutPrefix(c.Repo.URL, "oci://")
		// Append chart name to URL if not already present
		if !strings.HasSuffix(url, c.Name) {
			url = url + "/" + c.Name
		}
		vPrefix := strings.Contains(c.Version, "v")
		tags, err := c.RegistryClient.Tags(url)
		if err != nil {
			return "", err
		}

		vs := []semver.Version{}
		for _, t := range tags {
			s, err := semver.ParseTolerant(t)
			if err != nil {
				// non semver tag
				continue
			}
			vs = append(vs, s)
		}

		return latestStableVersion(vs, vPrefix)
	}

	indexPath := fmt.Sprintf("%s/%s-index.yaml", settings.RepositoryCache, c.Repo.Name)
	index, err := repo.LoadIndexFile(indexPath)
	if err != nil {
		return "", err
	}
	return latestStableVersion(chartVersionsToSemver(index.Entries[c.Name]), strings.Contains(c.Version, "v"))
}

type FunctionLoader struct {
	LoadFunc func(indexFilePath string) (*repo.IndexFile, error)
}

func (fl *FunctionLoader) LoadIndexFile(indexFilePath string) (*repo.IndexFile, error) {
	return fl.LoadFunc(indexFilePath)
}
