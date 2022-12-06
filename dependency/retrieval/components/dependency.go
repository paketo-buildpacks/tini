package components

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/paketo-buildpacks/packit/v2/cargo"
)

//go:generate faux --interface SignatureVerifier --output fakes/signature_verifier.go
type SignatureVerifier interface {
	Verify(signatureURL, targetURL string) error
}

func ConvertReleaseToDependency(release Release, signatureVerifier SignatureVerifier) (cargo.ConfigMetadataDependency, error) {
	var source, binary, binarySHA256, binaryASC ReleaseFile
	for _, f := range release.Files {
		if f.Name == "source" {
			source = f
		}

		if f.Name == "tini-static" {
			binary = f
		}

		if f.Name == "tini-static.sha256sum" {
			binarySHA256 = f
		}

		if f.Name == "tini-static.asc" {
			binaryASC = f
		}
	}

	if (source == ReleaseFile{} || binary == ReleaseFile{} || binarySHA256 == ReleaseFile{} || binaryASC == ReleaseFile{}) {
		return cargo.ConfigMetadataDependency{}, fmt.Errorf("required files are missing from the release")
	}

	// Obtain source sha256
	sourceResponse, err := http.Get(source.URL)
	if err != nil {
		return cargo.ConfigMetadataDependency{}, err
	}
	defer sourceResponse.Body.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, sourceResponse.Body); err != nil {
		return cargo.ConfigMetadataDependency{}, err
	}

	sourceChecksum := fmt.Sprintf("%x", hasher.Sum(nil))

	purl := GeneratePURL("tini", release.Version, sourceChecksum, source.URL)

	licenses, err := GenerateLicenseInformation(source.URL)
	if err != nil {
		return cargo.ConfigMetadataDependency{}, err
	}

	shasumResponse, err := http.Get(binarySHA256.URL)
	if err != nil {
		return cargo.ConfigMetadataDependency{}, err
	}
	defer shasumResponse.Body.Close()

	b, err := io.ReadAll(shasumResponse.Body)
	if err != nil {
		return cargo.ConfigMetadataDependency{}, err
	}

	split := strings.Split(strings.TrimSpace(string(b)), " ")
	if len(split) < 2 {
		return cargo.ConfigMetadataDependency{}, fmt.Errorf("unable to parse the sha256 file")
	}
	checksum := split[0]

	// Validate the artifact
	response, err := http.Get(binary.URL)
	if err != nil {
		return cargo.ConfigMetadataDependency{}, err
	}
	defer response.Body.Close()

	vr := cargo.NewValidatedReader(response.Body, fmt.Sprintf("sha256:%s", checksum))
	valid, err := vr.Valid()
	if err != nil {
		return cargo.ConfigMetadataDependency{}, err
	}

	if !valid {
		return cargo.ConfigMetadataDependency{}, fmt.Errorf("the given checksum of the artifact does not match with downloaded artifact")
	}

	err = signatureVerifier.Verify(binaryASC.URL, binary.URL)
	if err != nil {
		return cargo.ConfigMetadataDependency{}, err
	}

	return cargo.ConfigMetadataDependency{
		Checksum:       fmt.Sprintf("sha256:%s", checksum),
		ID:             "tini",
		Name:           "Tini",
		Version:        release.Version,
		Source:         source.URL,
		SourceChecksum: fmt.Sprintf("sha256:%s", sourceChecksum),
		CPE:            fmt.Sprintf(`cpe:2.3:a:tini_project:tini:%s:*:*:*:*:*:*:*`, release.Version),
		PURL:           purl,
		Licenses:       licenses,
		URI:            binary.URL,
	}, nil
}
