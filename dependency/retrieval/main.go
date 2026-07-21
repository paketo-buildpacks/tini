package main

import (
	"log"

	"github.com/Masterminds/semver/v3"
	"github.com/paketo-buildpacks/libdependency/versionology"

	"github.com/paketo-buildpacks/libdependency/retrieve"

	"github.com/paketo-buildpacks/tini/dependency/retrieval/components"
)

type TiniMetadata struct {
	SemverVersion *semver.Version
}

func (tiniMetadata TiniMetadata) Version() *semver.Version {
	return tiniMetadata.SemverVersion
}

func main() {
	retrieve.NewMetadataWithPlatforms("tini", getAllVersions, components.GenerateMetadataWithPlatform)
}

func getAllVersions() (versionology.VersionFetcherArray, error) {

	fetcher := components.NewFetcher()
	releases, err := fetcher.Get()
	if err != nil {
		log.Fatal(err)
	}

	var versions []versionology.VersionFetcher
	for _, release := range releases {
		semverVersion, _ := semver.NewVersion(release.Version)

		versions = append(versions, TiniMetadata{
			semverVersion,
		})
	}

	return versions, nil
}
