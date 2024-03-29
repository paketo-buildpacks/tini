package components_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnit(t *testing.T) {
	suite := spec.New("tini-retrieval", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Dependency", testDependency)
	suite("License", testLicense)
	suite("Output", testOutput)
	suite("Purl", testPurl)
	suite("Releases", testReleases)
	suite("Verifier", testVerifier)
	suite("Versions", testVersions)
	suite.Run(t)
}
