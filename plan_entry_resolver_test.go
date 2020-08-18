package tini_test

import (
	"bytes"
	"testing"

	"github.com/paketo-buildpacks/packit"
	tini "github.com/paketo-buildpacks/tini"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testPlanEntryResolver(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		buffer   *bytes.Buffer
		resolver tini.PlanEntryResolver
	)

	it.Before(func() {
		buffer = bytes.NewBuffer(nil)
		resolver = tini.NewPlanEntryResolver(tini.NewLogEmitter(buffer))
	})

	context("when entry flags differ", func() {
		context("OR's them together on best plan entry", func() {
			it("has all flags", func() {
				entry := resolver.Resolve([]packit.BuildpackPlanEntry{
					{
						Name: "tini",
						Metadata: map[string]interface{}{
							"launch": true,
						},
					},
					{
						Name: "tini",
						Metadata: map[string]interface{}{
							"build": true,
						},
					},
				})
				Expect(entry).To(Equal(packit.BuildpackPlanEntry{
					Name: "tini",
					Metadata: map[string]interface{}{
						"build":  true,
						"launch": true,
					},
				}))
			})
		})
	})
}
