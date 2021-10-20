package main

import (
	"os"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/cargo"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/draft"
	"github.com/paketo-buildpacks/packit/postal"
	"github.com/paketo-buildpacks/packit/scribe"
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
