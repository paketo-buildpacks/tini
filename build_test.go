package tini_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/sbom"

	//nolint Ignore SA1019, informed usage of deprecated package
	"github.com/paketo-buildpacks/packit/v2/paketosbom"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/paketo-buildpacks/tini"
	"github.com/paketo-buildpacks/tini/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir         string
		workingDir        string
		cnbDir            string
		entryResolver     *fakes.EntryResolver
		dependencyManager *fakes.DependencyManager
		sbomGenerator     *fakes.SBOMGenerator
		buffer            *bytes.Buffer

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		buffer = bytes.NewBuffer(nil)

		entryResolver = &fakes.EntryResolver{}
		entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{Name: "tini"}

		dependencyManager = &fakes.DependencyManager{}
		dependencyManager.ResolveCall.Returns.Dependency = postal.Dependency{
			ID:       "tini",
			Name:     "tini-dependency-name",
			Checksum: "sha256:tini-dependency-sha",
			Stacks:   []string{"some-stack"},
			URI:      "tini-dependency-uri",
			Version:  "tini-dependency-version",
		}
		dependencyManager.GenerateBillOfMaterialsCall.Returns.BOMEntrySlice = []packit.BOMEntry{
			{
				Name: "tini",
				Metadata: paketosbom.BOMMetadata{
					Version: "tini-dependency-version",
					Checksum: paketosbom.BOMChecksum{
						Algorithm: paketosbom.SHA256,
						Hash:      "tini-dependency-sha",
					},
					URI: "tini-dependency-uri",
				},
			},
		}

		sbomGenerator = &fakes.SBOMGenerator{}
		sbomGenerator.GenerateFromDependencyCall.Returns.SBOM = sbom.SBOM{}

		build = tini.Build(entryResolver, dependencyManager, chronos.DefaultClock, scribe.NewEmitter(buffer), sbomGenerator)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("returns a result that installs tini", func() {
		result, err := build(packit.BuildContext{
			WorkingDir: workingDir,
			CNBPath:    cnbDir,
			Stack:      "some-stack",
			BuildpackInfo: packit.BuildpackInfo{
				Name:        "Some Buildpack",
				Version:     "some-version",
				SBOMFormats: []string{sbom.CycloneDXFormat, sbom.SPDXFormat},
			},
			Platform: packit.Platform{Path: "platform"},
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{
						Name: "tini",
					},
				},
			},
			Layers: packit.Layers{Path: layersDir},
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(1))
		layer := result.Layers[0]

		Expect(layer.Name).To(Equal("tini"))
		Expect(layer.Path).To(Equal(filepath.Join(layersDir, "tini")))
		Expect(layer.Metadata).To(Equal(map[string]interface{}{
			"dependency-checksum": "sha256:tini-dependency-sha",
		}))

		Expect(layer.SBOM.Formats()).To(Equal([]packit.SBOMFormat{
			{
				Extension: sbom.Format(sbom.CycloneDXFormat).Extension(),
				Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.CycloneDXFormat),
			},
			{
				Extension: sbom.Format(sbom.SPDXFormat).Extension(),
				Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.SPDXFormat),
			},
		}))

		Expect(filepath.Join(layersDir, "tini")).To(BeADirectory())

		Expect(entryResolver.ResolveCall.Receives.Entries).To(Equal([]packit.BuildpackPlanEntry{
			{Name: "tini"},
		}))

		Expect(dependencyManager.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbDir, "buildpack.toml")))
		Expect(dependencyManager.ResolveCall.Receives.Id).To(Equal("tini"))
		Expect(dependencyManager.ResolveCall.Receives.Stack).To(Equal("some-stack"))

		Expect(dependencyManager.DeliverCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:       "tini",
			Name:     "tini",
			Checksum: "sha256:tini-dependency-sha",
			Stacks:   []string{"some-stack"},
			URI:      "tini-dependency-uri",
			Version:  "tini-dependency-version",
		}))
		Expect(dependencyManager.DeliverCall.Receives.CnbPath).To(Equal(cnbDir))
		Expect(dependencyManager.DeliverCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "tini", "bin")))
		Expect(dependencyManager.DeliverCall.Receives.PlatformPath).To(Equal("platform"))

		Expect(dependencyManager.GenerateBillOfMaterialsCall.Receives.Dependencies).To(Equal([]postal.Dependency{
			{
				ID:       "tini",
				Name:     "tini-dependency-name",
				Checksum: "sha256:tini-dependency-sha",
				Stacks:   []string{"some-stack"},
				URI:      "tini-dependency-uri",
				Version:  "tini-dependency-version",
			},
		}))

		Expect(sbomGenerator.GenerateFromDependencyCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:       "tini",
			Name:     "tini-dependency-name",
			Checksum: "sha256:tini-dependency-sha",
			Stacks:   []string{"some-stack"},
			URI:      "tini-dependency-uri",
			Version:  "tini-dependency-version",
		}))
		Expect(sbomGenerator.GenerateFromDependencyCall.Receives.Dir).To(Equal(filepath.Join(layersDir, "tini")))

		Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
		Expect(buffer.String()).To(ContainSubstring("Executing build process"))
	})

	context("when the build plan entry includes the build, launch flag", func() {
		it.Before(func() {
			entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
				Name: "tini",
				Metadata: map[string]interface{}{
					"launch": true,
					"build":  true,
				},
			}

			entryResolver.MergeLayerTypesCall.Returns.Launch = true
			entryResolver.MergeLayerTypesCall.Returns.Build = true
		})

		it("marks the tini layer as build, cache and launch", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Stack:      "some-stack",
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "tini",
							Metadata: map[string]interface{}{
								"launch": true,
								"build":  true,
							},
						},
					},
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(1))
			layer := result.Layers[0]

			Expect(layer.Name).To(Equal("tini"))
			Expect(layer.Path).To(Equal(filepath.Join(layersDir, "tini")))
			Expect(layer.Metadata).To(Equal(map[string]interface{}{
				"dependency-checksum": "sha256:tini-dependency-sha",
			}))

			Expect(layer.Build).To(BeTrue())
			Expect(layer.Cache).To(BeTrue())
			Expect(layer.Launch).To(BeTrue())

			Expect(result.Build.BOM).To(HaveLen(1))
			buildBOMEntry := result.Build.BOM[0]
			Expect(buildBOMEntry.Name).To(Equal("tini"))
			Expect(buildBOMEntry.Metadata).To(Equal(paketosbom.BOMMetadata{
				Version: "tini-dependency-version",
				Checksum: paketosbom.BOMChecksum{
					Algorithm: paketosbom.SHA256,
					Hash:      "tini-dependency-sha",
				},
				URI: "tini-dependency-uri",
			}))

			Expect(result.Launch.BOM).To(HaveLen(1))
			launchBOMEntry := result.Launch.BOM[0]
			Expect(launchBOMEntry.Name).To(Equal("tini"))
			Expect(launchBOMEntry.Metadata).To(Equal(paketosbom.BOMMetadata{
				Version: "tini-dependency-version",
				Checksum: paketosbom.BOMChecksum{
					Algorithm: paketosbom.SHA256,
					Hash:      "tini-dependency-sha",
				},
				URI: "tini-dependency-uri",
			}))
		})
	})

	context("when the layer metadata contains a cache match", func() {
		it.Before(func() {
			err := os.WriteFile(filepath.Join(layersDir, "tini.toml"), []byte(`
[metadata]
	dependency-checksum = "sha256:tini-dependency-sha"
`), 0600)
			Expect(err).NotTo(HaveOccurred())

			entryResolver.MergeLayerTypesCall.Returns.Launch = false
			entryResolver.MergeLayerTypesCall.Returns.Build = true
		})

		it("reuses the existing layer", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Stack:      "some-stack",
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Platform: packit.Platform{Path: "platform"},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "tini",
						},
					},
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(1))
			layer := result.Layers[0]

			Expect(layer.Name).To(Equal("tini"))
			Expect(layer.Path).To(Equal(filepath.Join(layersDir, "tini")))
			Expect(layer.Metadata).To(Equal(map[string]interface{}{
				"dependency-checksum": "sha256:tini-dependency-sha",
			}))

			Expect(dependencyManager.DeliverCall.CallCount).To(Equal(0))
			Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
			Expect(buffer.String()).To(ContainSubstring("Reusing cached layer"))
		})
	})

	context("failure cases", func() {
		context("when the dependency cannot be resolved", func() {
			it.Before(func() {
				dependencyManager.ResolveCall.Returns.Error = errors.New("failed to resolve dependency")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "tini"},
						},
					},
					Layers: packit.Layers{Path: layersDir},
					Stack:  "some-stack",
				})
				Expect(err).To(MatchError("failed to resolve dependency"))
			})
		})

		context("when the layer cannot be retrieved", func() {
			it.Before(func() {
				err := os.WriteFile(filepath.Join(layersDir, "tini.toml"), nil, 0000)
				Expect(err).NotTo(HaveOccurred())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "tini"},
						},
					},
					Layers: packit.Layers{Path: layersDir},
					Stack:  "some-stack",
				})
				Expect(err).To(MatchError(ContainSubstring("failed to parse layer content metadata")))
			})
		})

		context("when the layer cannot be reset", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(layersDir, "tini", "something"), os.ModePerm)).To(Succeed())
				Expect(os.Chmod(filepath.Join(layersDir, "tini"), 0500)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(filepath.Join(layersDir, "tini"), os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "tini"},
						},
					},
					Layers: packit.Layers{Path: layersDir},
					Stack:  "some-stack",
				})
				Expect(err).To(MatchError(ContainSubstring("could not remove file")))
			})
		})

		context("when the dependency cannot be installed", func() {
			it.Before(func() {
				dependencyManager.DeliverCall.Returns.Error = errors.New("failed to install dependency")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "tini"},
						},
					},
					Layers: packit.Layers{Path: layersDir},
					Stack:  "some-stack",
				})
				Expect(err).To(MatchError("failed to install dependency"))
			})
		})

		context("when the layers directory cannot be written to", func() {
			it.Before(func() {
				Expect(os.Chmod(layersDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(layersDir, os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "tini"},
						},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})
		context("when generating the SBOM returns an error", func() {
			it.Before(func() {
				sbomGenerator.GenerateFromDependencyCall.Returns.Error = errors.New("failed to generate SBOM")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "tini"},
						},
					},
					Layers: packit.Layers{Path: layersDir},
					Stack:  "some-stack",
				})
				Expect(err).To(MatchError(ContainSubstring("failed to generate SBOM")))
			})
		})

		context("when formatting the SBOM returns an error", func() {
			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					BuildpackInfo: packit.BuildpackInfo{SBOMFormats: []string{"random-format"}},
					CNBPath:       cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "tini"},
						},
					},
					Layers: packit.Layers{Path: layersDir},
					Stack:  "some-stack",
				})
				Expect(err).To(MatchError("unsupported SBOM format: 'random-format'"))
			})
		})
	})
}
