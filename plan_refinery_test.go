package tini_test

import (
	"testing"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/postal"
	tini "github.com/paketo-buildpacks/tini"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testPlanRefinery(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		refinery tini.PlanRefinery
	)

	it.Before(func() {
		refinery = tini.NewPlanRefinery()
	})

	context("BillOfMaterials", func() {
		it("returns a refined build plan entry", func() {
			entry := refinery.BillOfMaterials(postal.Dependency{
				ID:      "some-id",
				Name:    "some-name",
				Stacks:  []string{"some-stack"},
				URI:     "some-uri",
				SHA256:  "some-sha",
				Version: "some-version",
			})
			Expect(entry).To(Equal(packit.BuildpackPlanEntry{
				Name:    "some-id",
				Version: "some-version",
				Metadata: map[string]interface{}{
					"licenses": []string{},
					"name":     "some-name",
					"sha256":   "some-sha",
					"stacks":   []string{"some-stack"},
					"uri":      "some-uri",
				},
			}))
		})
	})
}
