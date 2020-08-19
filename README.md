# Tini Cloud Native Buildpack

The Tini CNB provides the [Tini](https://github.com/krallin/tini) executable.
The buildpack installs tini onto the `$PATH` which makes it available for
subsequent buildpacks and/or the final container image.

## Integration

The Tini CNB provides `tini` as a dependency. Downstream
buildpacks can require the tini dependency by generating a [Build Plan
TOML](https://github.com/buildpacks/spec/blob/master/buildpack.md#build-plan-toml)
file that looks like the following:

```toml
[[requires]]

  # The name of the Tini dependency is "tini". This value is
  # considered part of the public API for the buildpack and will not change
  # without a plan for deprecation.
  name = "tini"

  # Note: The version field is unsupported at this time

  # The Tini buildpack supports some non-required metadata options.
  [requires.metadata]

    # Setting the build flag to true will ensure that the Tini
    # dependency is available on the $PATH for subsequent buildpacks during
    # their build phase. If you are writing a buildpack that needs to run Tini
    # during its build process, this flag should be set to true.
    build = true

    # Setting the launch flag to true will ensure that the Tini
    # dependency is available on the $PATH for the running application. If you are
    # writing an application that needs to run Tini at runtime, this flag should
    # be set to true.
    launch = true
```

## Usage

To package this buildpack for consumption:
```
$ ./scripts/package.sh
```

## `buildpack.yml` Configuration

The tini buildpack does not support configurations via `buildpack.yml`.
