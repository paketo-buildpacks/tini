package main

import (
	"os"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/paketo-buildpacks/tini"
)

func main() {
	packit.Run(
		tini.Detect(),
		tini.Build(
			draft.NewPlanner(),
			postal.NewService(cargo.NewTransport()),
			chronos.DefaultClock,
			scribe.NewEmitter(os.Stdout),
		),
	)
}
