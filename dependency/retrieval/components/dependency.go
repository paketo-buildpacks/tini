package components

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/paketo-buildpacks/libdependency/retrieve"
	"github.com/paketo-buildpacks/libdependency/versionology"
	"github.com/paketo-buildpacks/packit/v2/cargo"
)

func ConvertReleaseToDependency(release Release, platform cargo.ConfigTarget) ([]versionology.Dependency, error) {
	var source, binary, binarySHA256, binaryASC ReleaseFile
	for _, f := range release.Files {
		if f.Name == "source" {
			source = f
		}

		if f.Name == fmt.Sprintf("tini-static-%s", platform.Arch) {
			binary = f
		}

		if f.Name == fmt.Sprintf("tini-static-%s.sha256sum", platform.Arch) {
			binarySHA256 = f
		}

		if f.Name == fmt.Sprintf("tini-static-%s.asc", platform.Arch) {
			binaryASC = f
		}
	}

	if (source == ReleaseFile{} || binary == ReleaseFile{} || binarySHA256 == ReleaseFile{} || binaryASC == ReleaseFile{}) {
		return nil, fmt.Errorf("required files are missing from the release")
	}

	// Obtain source sha256
	sourceResponse, err := http.Get(source.URL)
	if err != nil {
		return nil, err
	}
	defer sourceResponse.Body.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, sourceResponse.Body); err != nil {
		return nil, err
	}

	sourceChecksum := fmt.Sprintf("%x", hasher.Sum(nil))

	purl := GeneratePURL("tini", release.Version, sourceChecksum, source.URL)

	licenses, err := GenerateLicenseInformation(source.URL)
	if err != nil {
		return nil, err
	}

	shasumResponse, err := http.Get(binarySHA256.URL)
	if err != nil {
		return nil, err
	}
	defer shasumResponse.Body.Close()

	b, err := io.ReadAll(shasumResponse.Body)
	if err != nil {
		return nil, err
	}

	split := strings.Split(strings.TrimSpace(string(b)), " ")
	if len(split) < 2 {
		return nil, fmt.Errorf("unable to parse the sha256 file")
	}
	checksum := split[0]

	// Validate the artifact
	response, err := http.Get(binary.URL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	vr := cargo.NewValidatedReader(response.Body, fmt.Sprintf("sha256:%s", checksum))
	valid, err := vr.Valid()
	if err != nil {
		return nil, err
	}

	if !valid {
		return nil, fmt.Errorf("the given checksum of the artifact does not match with downloaded artifact")
	}

	signatureVerifier := NewVerifier()
	err = signatureVerifier.Verify(binaryASC.URL, binary.URL)
	if err != nil {
		return nil, err
	}

	dep := cargo.ConfigMetadataDependency{
		Arch:           platform.Arch,
		Checksum:       fmt.Sprintf("sha256:%s", checksum),
		ID:             "tini",
		Name:           "Tini",
		Version:        release.Version,
		Source:         source.URL,
		SourceChecksum: fmt.Sprintf("sha256:%s", sourceChecksum),
		CPE:            fmt.Sprintf(`cpe:2.3:a:tini_project:tini:%s:*:*:*:*:*:*:*`, release.Version),
		OS:             platform.OS,
		PURL:           purl,
		Licenses:       licenses,
		URI:            binary.URL,
	}

	allStacksDependency, err := versionology.NewDependency(dep, "*")
	if err != nil {
		return nil, fmt.Errorf("could not get create * dependency: %w", err)
	}

	return []versionology.Dependency{allStacksDependency}, nil
}

func GenerateMetadataWithPlatform(versionFetcher versionology.VersionFetcher, platform retrieve.Platform) ([]versionology.Dependency, error) {
	version := versionFetcher.Version().String()

	fetcher := NewFetcher()
	releases, err := fetcher.Get()
	if err != nil {
		return nil, err
	}

	var allStacksDependency versionology.Dependency
	for _, release := range releases {
		if release.Version == version {
			return ConvertReleaseToDependency(release, cargo.ConfigTarget{
				OS:   platform.OS,
				Arch: platform.Arch,
			})
		}
	}

	return []versionology.Dependency{allStacksDependency}, nil

}
