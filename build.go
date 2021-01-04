package tini

import (
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/postal"
)

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
type EntryResolver interface {
	Resolve([]packit.BuildpackPlanEntry) packit.BuildpackPlanEntry
}

//go:generate faux --interface DependencyManager --output fakes/dependency_manager.go
type DependencyManager interface {
	Resolve(path, id, version, stack string) (postal.Dependency, error)
	Install(dependency postal.Dependency, cnbPath, layerPath string) error
}

//go:generate faux --interface BuildPlanRefinery --output fakes/build_plan_refinery.go
type BuildPlanRefinery interface {
	BillOfMaterials(postal.Dependency) packit.BuildpackPlanEntry
}

func Build(
	entries EntryResolver,
	dependencies DependencyManager,
	planRefinery BuildPlanRefinery,
	clock chronos.Clock,
	logger LogEmitter,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		entry := entries.Resolve(context.Plan.Entries)

		version, ok := entry.Metadata["version"].(string)
		if !ok {
			version = "default"
		}

		dependency, err := dependencies.Resolve(
			filepath.Join(context.CNBPath, "buildpack.toml"),
			entry.Name,
			version,
			context.Stack)
		if err != nil {
			return packit.BuildResult{}, err
		}

		bom := planRefinery.BillOfMaterials(dependency)

		tiniLayer, err := context.Layers.Get(Tini)
		if err != nil {
			return packit.BuildResult{}, err
		}

		cachedSHA, ok := tiniLayer.Metadata[DependencyCacheKey].(string)
		if ok && cachedSHA == dependency.SHA256 {
			logger.Process("Reusing cached layer %s", tiniLayer.Path)
			logger.Break()

			return packit.BuildResult{
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{bom},
				},
				Layers: []packit.Layer{tiniLayer},
			}, nil
		}

		logger.Process("Executing build process")

		tiniLayer, err = tiniLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		tiniLayer.Launch = entry.Metadata["launch"] == true
		tiniLayer.Build = entry.Metadata["build"] == true
		tiniLayer.Cache = entry.Metadata["build"] == true

		logger.Subprocess("Installing Tini %s", dependency.Version)

		duration, err := clock.Measure(func() error {
			return dependencies.Install(dependency, context.CNBPath, tiniLayer.Path)
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		tiniLayer.Metadata = map[string]interface{}{
			DependencyCacheKey: dependency.SHA256,
			"built_at":         clock.Now().Format(time.RFC3339Nano),
		}

		return packit.BuildResult{
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{bom},
			},
			Layers: []packit.Layer{
				tiniLayer,
			},
		}, nil
	}
}
