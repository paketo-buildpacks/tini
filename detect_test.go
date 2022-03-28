package tini_test

import (
	"testing"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/tini"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
		detect packit.DetectFunc
	)

	it.Before(func() {
		detect = tini.Detect()
	})

	context("returns a plan that provides tini", func() {
		detect = tini.Detect()
		it("detects", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: "/workspace",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: "tini"},
				},
			}))
		})
	})
}
