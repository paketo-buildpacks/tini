package main

import (
	"flag"
	"log"
	"os"

	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/tini/dependency/retrieval/components"
)

var targetMap = map[string][]string{
	"": []string{"io.buildpacks.stacks.jammy", "io.buildpacks.stacks.bionic"},
}

func main() {
	var buildpackTOMLPath, outputPath string
	set := flag.NewFlagSet("", flag.ContinueOnError)
	set.StringVar(&buildpackTOMLPath, "buildpack-toml-path", "", "path to the buildpack.toml file")
	set.StringVar(&outputPath, "output", "", "path to the output file")
	err := set.Parse(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	fetcher := components.NewFetcher()
	releases, err := fetcher.Get()
	if err != nil {
		log.Fatal(err)
	}

	var versions []string
	for _, release := range releases {
		versions = append(versions, release.SemVer.String())
	}

	newVersions, err := components.FindNewVersions(buildpackTOMLPath, versions)
	if err != nil {
		log.Fatal(err)
	}

	verifier := components.NewVerifier()

	var dependencies []cargo.ConfigMetadataDependency
	for _, version := range newVersions {
		for _, r := range releases {
			if r.SemVer.String() == version {
				dependency, err := components.ConvertReleaseToDependency(r, verifier)
				if err != nil {
					log.Fatal(err)
				}
				dependencies = append(dependencies, dependency)
			}
		}
	}

	err = components.WriteOutput(outputPath, dependencies, targetMap)
	if err != nil {
		log.Fatal(err)
	}
}
