package tini_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/postal"
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
		timestamp         time.Time
		entryResolver     *fakes.EntryResolver
		dependencyManager *fakes.DependencyManager
		planRefinery      *fakes.BuildPlanRefinery
		buffer            *bytes.Buffer

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = ioutil.TempDir("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = ioutil.TempDir("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		buffer = bytes.NewBuffer(nil)
		logEmitter := tini.NewLogEmitter(buffer)

		timestamp = time.Now()
		clock := chronos.NewClock(func() time.Time {
			return timestamp
		})

		entryResolver = &fakes.EntryResolver{}
		entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
			Name: "tini",
		}

		dependencyManager = &fakes.DependencyManager{}
		dependencyManager.ResolveCall.Returns.Dependency = postal.Dependency{
			ID:      "tini",
			Name:    "tini-dependency-name",
			SHA256:  "tini-dependency-sha",
			Stacks:  []string{"some-stack"},
			URI:     "tini-dependency-uri",
			Version: "tini-dependency-version",
		}

		planRefinery = &fakes.BuildPlanRefinery{}
		planRefinery.BillOfMaterialsCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
			Name: "tini",
			Metadata: map[string]interface{}{
				"name":   "tini-dependency-name",
				"sha256": "tini-dependency-sha",
				"stacks": []string{"some-stack"},
				"uri":    "tini-dependency-uri",
			},
		}
		build = tini.Build(entryResolver, dependencyManager, planRefinery, clock, logEmitter)
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
				Name:    "Some Buildpack",
				Version: "some-version",
			},
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

		Expect(result).To(Equal(packit.BuildResult{
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{
						Name: "tini",
						Metadata: map[string]interface{}{
							"name":   "tini-dependency-name",
							"sha256": "tini-dependency-sha",
							"stacks": []string{"some-stack"},
							"uri":    "tini-dependency-uri",
						},
					},
				},
			},
			Layers: []packit.Layer{
				{
					Name:             "tini",
					Path:             filepath.Join(layersDir, "tini"),
					SharedEnv:        packit.Environment{},
					BuildEnv:         packit.Environment{},
					LaunchEnv:        packit.Environment{},
					ProcessLaunchEnv: map[string]packit.Environment{},
					Build:            false,
					Launch:           false,
					Cache:            false,
					Metadata: map[string]interface{}{
						tini.DependencyCacheKey: "tini-dependency-sha",
						"built_at":              timestamp.Format(time.RFC3339Nano),
					},
				},
			},
		}))

		Expect(filepath.Join(layersDir, "tini")).To(BeADirectory())

		Expect(entryResolver.ResolveCall.Receives.BuildpackPlanEntrySlice).To(Equal([]packit.BuildpackPlanEntry{
			{
				Name: "tini",
			},
		}))

		Expect(dependencyManager.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbDir, "buildpack.toml")))
		Expect(dependencyManager.ResolveCall.Receives.Id).To(Equal("tini"))
		Expect(dependencyManager.ResolveCall.Receives.Stack).To(Equal("some-stack"))

		Expect(dependencyManager.InstallCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:      "tini",
			Name:    "tini-dependency-name",
			SHA256:  "tini-dependency-sha",
			Stacks:  []string{"some-stack"},
			URI:     "tini-dependency-uri",
			Version: "tini-dependency-version",
		}))
		Expect(dependencyManager.InstallCall.Receives.CnbPath).To(Equal(cnbDir))
		Expect(dependencyManager.InstallCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "tini")))

		Expect(planRefinery.BillOfMaterialsCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:      "tini",
			Name:    "tini-dependency-name",
			SHA256:  "tini-dependency-sha",
			Stacks:  []string{"some-stack"},
			URI:     "tini-dependency-uri",
			Version: "tini-dependency-version",
		}))

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

			Expect(result).To(Equal(packit.BuildResult{
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "tini",
							Metadata: map[string]interface{}{
								"name":   "tini-dependency-name",
								"sha256": "tini-dependency-sha",
								"stacks": []string{"some-stack"},
								"uri":    "tini-dependency-uri",
							},
						},
					},
				},
				Layers: []packit.Layer{
					{
						Name:             "tini",
						Path:             filepath.Join(layersDir, "tini"),
						SharedEnv:        packit.Environment{},
						BuildEnv:         packit.Environment{},
						LaunchEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Build:            true,
						Launch:           true,
						Cache:            true,
						Metadata: map[string]interface{}{
							tini.DependencyCacheKey: "tini-dependency-sha",
							"built_at":              timestamp.Format(time.RFC3339Nano),
						},
					},
				},
			}))
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

		context("when the dependency cannot be installed", func() {
			it.Before(func() {
				dependencyManager.InstallCall.Returns.Error = errors.New("failed to install dependency")
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
	})
}
