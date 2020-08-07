package main

import (
	"os"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/cargo"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/postal"
	"github.com/paketo-buildpacks/tini"
)

func main() {
	logEmitter := tini.NewLogEmitter(os.Stdout)
	entryResolver := tini.NewPlanEntryResolver(logEmitter)
	dependencyManager := postal.NewService(cargo.NewTransport())
	planRefinery := tini.NewPlanRefinery()

	packit.Run(
		tini.Detect(),
		tini.Build(
			entryResolver,
			dependencyManager,
			planRefinery,
			chronos.DefaultClock,
			logEmitter,
		),
	)
}
