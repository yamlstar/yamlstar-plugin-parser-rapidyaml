# YAMLStar RapidYAML Parser

An optional RapidYAML parser for YAMLStar, implemented with Go and C++.
The plugin produces YEBP v1 binary events and contains no Glojure runtime.
It requires a YAMLStar host supporting binary shared parser plugins.

```bash
yaml --parser=rapidyaml
yaml --plugin=parser=rapidyaml
yaml --parser=rapidyaml@0.1.0
```

The implementation name is `rapidyaml` and the plugin API is `parser`.
The repository and Python distribution are `yamlstar-plugin-parser-rapidyaml`.
Libraries retain the loader's standard name, `libyamlstar-plugin-rapidyaml.so`
on Linux and `.dylib` on macOS.
No parser-specific options are supported yet.

## Build and test

```bash
make build
make test
make test-conformance
make fuzz
make benchmark
```

Makes installs project tools into `.cache`.
A platform C/C++ compiler is required for cgo (GCC on Linux, Clang on macOS).
Run project-dependent commands with `make shell CMD='...'`.
The build pins rapidyaml v0.16.0 at
`f8ac8dd50f4f7916579d55a05ebf9c6488e52670`.
This checkout includes the c4core v0.6.0 source snapshot; it has no submodules.
The Go encoder is `yaml-events-binary-protocol v0.1.0`.

To try a local build from this directory:

```bash
YAMLSTAR_LIBRARY_PATH="$PWD/lib" yaml --parser=rapidyaml --no-plugin-install \
  --eval 'a: [1, 2, 3]'
```

## Python

```bash
pip install yamlstar-plugin-parser-rapidyaml
```

```python
import yamlstar

opts = yamlstar.Options().plugin(
    yamlstar.parser('rapidyaml', version='0.1.0'))
ys = yamlstar.YAMLStar(opts)
assert ys.load('a: [1, 2, 3]') == {'a': [1, 2, 3]}
```

Wheels expose the `rapidyaml` entry point in `yamlstar.plugins`.
They contain both default and exact-version library paths.
Other bindings select the same parser through their existing options mapping.
Shared plugins require the Glojure host; JVM, pure Go, and GraalVM hosts do not
load this plugin.

## Conformance

`make test-conformance` compares binary events against 402 fixtures in the
YAML test suite at `6ad3d2c62885d82fc349026c136ef560838fdf3d`.
The baseline records nine upstream tab-separation differences: three valid
inputs rejected, three invalid inputs accepted, and three event mismatches.
See `test/expected-events.txt` for individual cases and explanations.
New differences and unexpected improvements fail the test for review.
The adapter does not fall back to another parser.

Tests cover scalar styles, tags, directives, explicit document markers,
anchors, aliases, buffer growth, malformed input, ownership, and concurrent
calls through the C ABI.
Input buffers are limited to less than 512 MiB by the integer-event adapter.
Output is owned by the plugin and released with `yamlstar_plugin_v1_free`.
Status 0 returns YEBP, status 1 reports parse/configuration errors, and status 2
reports ABI/internal errors.
Metadata and errors use EDN, but successful events never do.

## Performance

The initial full CLI comparison is slower than built-in `go-yaml` on the
three measured fixtures.
See [Performance.md](Performance.md) for results and reproduction commands.
Binary transport avoids EDN serialization but does not guarantee that an
external parser beats the built-in parser.

## Release

```bash
make dist
make wheel
make test-wheel
make release v=0.1.0 d=1
make release v=0.1.0
```

Release archives use
`yamlstar-plugin-rapidyaml-VERSION-PLATFORM.tar.xz`.
The release workflow builds Linux x64/aarch64 with a manylinux 2.28 baseline,
and macOS x64/arm64, validates archives and wheels, and publishes checksums.
The repository must exist under `yamlstar/yamlstar-plugin-parser-rapidyaml`.
Configure PyPI trusted publishing for distribution
`yamlstar-plugin-parser-rapidyaml`, workflow `release.yaml`, environment `pypi`.
Release from `main`, or pass `a=1` to release from another branch.
No release or publication happens during normal builds or tests.

The integration workflow uses the YAMLStar branch selected by the repository
variable `YAMLSTAR_REF`, defaulting to `parser-rapidyaml` until integration is
merged into main.

Third-party notices are included in `licenses/` and release artifacts.
