package components_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/paketo-buildpacks/tini/dependency/retrieval/components"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testReleases(t *testing.T, context spec.G, it spec.S) {

	var (
		Expect = NewWithT(t).Expect
	)

	context("Fetcher", func() {
		var (
			fetcher components.Fetcher

			server *httptest.Server
		)

		it.Before(func() {
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodHead {
					http.Error(w, "NotFound", http.StatusNotFound)
					return
				}

				switch req.URL.Path {
				case "/repos/krallin/tini/releases":
					switch req.URL.RawQuery {
					case "per_page=100&page=1":
						w.WriteHeader(http.StatusOK)
						fmt.Fprintln(w, `[
  {
		"tag_name": "v0.19.0",
    "name": "v0.19.0",
    "draft": false,
    "prerelease": false,
    "tarball_url": "https://api.github.com/repos/krallin/tini/tarball/v0.19.0",
    "assets": [
      {
        "name": "tini-static",
        "browser_download_url": "https://github.com/krallin/tini/releases/download/v0.19.0/tini-static"
      }
    ]
  },
  {
    "tag_name": "cldr/2022-10-11",
    "name": "cldr/2022-10-11",
    "tarball_url": "draft source",
    "draft": false,
    "prerelease": true
  }
]`)
					case "per_page=100&page=2":
						w.WriteHeader(http.StatusOK)
						fmt.Fprintln(w, `[
  {
		"tag_name": "v0.18.0",
    "name": "v0.18.0",
    "draft": false,
    "prerelease": false,
    "tarball_url": "https://api.github.com/repos/krallin/tini/tarball/v0.18.0",
		"assets": [
      {
        "name": "tini-static",
        "browser_download_url": "https://github.com/krallin/tini/releases/download/v0.18.0/tini-static"
      }
    ]
  },
  {
    "tag_name": "cldr/2022-8-12",
    "name": "cldr/2022-8-12",
    "draft": true,
    "prerelease": false
  }
]`)
					case "per_page=100&page=3":
						w.WriteHeader(http.StatusOK)
						fmt.Fprintln(w, `[
]`)
					}

				case "/non-200/repos/krallin/tini/releases":
					w.WriteHeader(http.StatusTeapot)

				case "/no-parse/repos/krallin/tini/releases":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, `???`)

				case "/bad-version/repos/krallin/tini/releases":
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, `[
  {
    "name": "Tini invalid version"
  }
]`)

				default:
					t.Fatalf("unknown path: %s", req.URL.Path)
				}
			}))

			fetcher = components.NewFetcher().WithAPI(server.URL)
		})

		it("fetches a list of relevant releases", func() {
			releases, err := fetcher.Get()
			Expect(err).NotTo(HaveOccurred())

			Expect(releases).To(Equal([]components.Release{
				{
					SemVer:  semver.MustParse("0.19.0"),
					Version: "0.19.0",
					Files: []components.ReleaseFile{
						{
							Name: "tini-static",
							URL:  "https://github.com/krallin/tini/releases/download/v0.19.0/tini-static",
						},
						{
							Name: "source",
							URL:  "https://api.github.com/repos/krallin/tini/tarball/v0.19.0",
						},
					},
				},
				{
					SemVer:  semver.MustParse("0.18.0"),
					Version: "0.18.0",
					Files: []components.ReleaseFile{
						{
							Name: "tini-static",
							URL:  "https://github.com/krallin/tini/releases/download/v0.18.0/tini-static",
						},
						{
							Name: "source",
							URL:  "https://api.github.com/repos/krallin/tini/tarball/v0.18.0",
						},
					},
				},
			}))
		})

		context("failure cases", func() {
			context("when the release page get fails", func() {
				it.Before(func() {
					fetcher = fetcher.WithAPI("not a valid URL")
				})

				it("returns an error", func() {
					_, err := fetcher.Get()
					Expect(err).To(MatchError(ContainSubstring("unsupported protocol scheme")))
				})
			})

			context("when the release page returns non 200 code", func() {
				it.Before(func() {
					fetcher = fetcher.WithAPI(fmt.Sprintf("%s/non-200", server.URL))
				})

				it("returns an error", func() {
					_, err := fetcher.Get()
					Expect(err).To(MatchError("received a non 200 status code: status code 418 received"))
				})
			})

			context("when the release page cannot parse", func() {
				it.Before(func() {
					fetcher = fetcher.WithAPI(fmt.Sprintf("%s/no-parse", server.URL))
				})

				it("returns an error", func() {
					_, err := fetcher.Get()
					Expect(err).To(MatchError(ContainSubstring("invalid character '?' looking for beginning of value")))
				})
			})

			context("when the version is unparsable", func() {
				it.Before(func() {
					fetcher = fetcher.WithAPI(fmt.Sprintf("%s/bad-version", server.URL))
				})

				it("returns an error", func() {
					_, err := fetcher.Get()
					Expect(err).To(MatchError(ContainSubstring("Invalid Semantic Version")))
				})
			})
		})
	})
}
