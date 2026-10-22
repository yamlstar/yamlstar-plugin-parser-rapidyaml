# YAMLStar RapidYAML Parser

An optional RapidYAML parser for YAMLStar, implemented with Go and C++.
The plugin implements ABI v2 transforms returning EDN parser events.
It contains no Glojure runtime.
It requires a YAMLStar host with the `parser=rapidyaml` ABI v2 registration.
Release 0.1.0 used ABI v1 and cannot be loaded by the new host.

```bash
yaml --plugin=parser=rapidyaml
```

The implementation name is `rapidyaml` and the plugin API is `parser`.
The repository and Python distribution are `yamlstar-plugin-parser-rapidyaml`.
Libraries use `libyamlstar-plugin-parser-rapidyaml.so` on Linux and
`.dylib` on macOS.
The native parser accepts no parser-specific options.
The Go adapter also accepts the installation option described below.

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

## go-yaml integration

The `goyaml` package exports `New(options ...Option)` and `Register() error`
for go-yaml.
The adapter loads this plugin's ABI v2 shared library at runtime and requires
no cgo in the consuming Go executable.
Linux, FreeBSD, and macOS hosts can load manually installed libraries.
Automatic installation supports the four Linux and macOS release platforms.

```go
import (
    "github.com/yamlstar/yamlstar-plugin-parser-rapidyaml/goyaml"
    "go.yaml.in/yaml/v4"
)

var value any
err := yaml.Load([]byte("answer: 42"), &value,
    yaml.WithPlugin(goyaml.New()))
```

Call `goyaml.Register()` during startup before parsing named configuration.
It registers `yaml-parser=rapidyaml` at `goyaml.Version`, preserving the native
Go parser as the default.
Registration and `--help` do not load or download a library.
The library is resolved on the first parse with RapidYAML.
The compiled-in `go-yaml` parser requires no parser shared library.

The adapter requires the native library version matching its release.
If missing, it downloads that exact release's archive and `SHA256SUMS` from
GitHub, verifies them, and installs the library atomically.
This requires no YAMLStar executable or external download/archive commands.
Existing incompatible libraries produce an error and are preserved.
Unknown configuration options, duplicate registration, and mismatched requested
versions also return errors.

Use `goyaml.New(goyaml.WithInstall(false))` to disable downloads in Go code.
For named configuration, use:

```yaml
plugin:
  yaml-parser:
    name: rapidyaml
    install: false
```

`YAMLSTAR_LIBRARY_PATH` specifies library search directories.
When unset, the adapter searches beside the executable, its `../lib` directory,
`$HOME/.local/lib`, `/usr/local/lib`, and `/usr/lib`.
An explicitly configured path replaces those defaults, including an empty path.
Missing libraries are installed into the first writable search directory.
Select another directory with `YAMLSTAR_LIBRARY_PATH` when an incompatible
library already exists.

After publishing the native `0.1.2` release assets and Go module tag `v0.1.2`,
build a go-yaml CLI once:

```bash
make cli \
  PLUGIN-PACKAGE=github.com/yamlstar/yamlstar-plugin-parser-rapidyaml/goyaml@v0.1.2
./go-yaml --plugin=yaml-parser=rapidyaml -j < input.yaml
./go-yaml --plugin=yaml-parser=go-yaml -j < input.yaml
```

`PLUGIN-PACKAGE` links the adapter; runtime `--plugin` selects it.
The packages-only build retains native defaults.
To select RapidYAML as the build default, add `PLUGIN=yaml-parser=rapidyaml`.
No parser switch requires another build.
The Go module tag and native release tag refer to the same commit.
Archives retain unprefixed versions in their names and download URLs.

`make test-goyaml` tests the adapter with cgo disabled.
`make test-goyaml-race` runs its concurrency and installation tests with the
Go race detector.
Set `GO_YAML_CLI` to an adapter-enabled executable to run CLI regressions too.
Installer tests use local HTTP fixtures without contacting GitHub.
`make test-goyaml-cli` uses `GO_YAML_DIR` (default `../go-yaml`) to test the
actual generic `PLUGIN-PACKAGE` build with a local adapter snapshot.
It stages sources, a file module proxy, and its executable under
`.cache/integration`, leaving the supplied checkout and executable unchanged.

To try a local build from this directory:

```bash
make build
YAMLSTAR_LIBRARY_PATH="$PWD/lib" yaml --plugin=parser=rapidyaml --no-install \
  --eval 'a: [1, 2, 3]'
```

YAMLStar and go-yaml can share that same library directory.
For this coordinated workspace, build YAMLStar's CLI and shared host library
from the workspace root with `make cli build-libyamlstar`, then build go-yaml
with the wanted adapter packages once.
Until the adapters are published, stage local module snapshots under `.cache`
with an isolated file Go proxy and module cache, and supply their explicit
versions through `PLUGIN-PACKAGE`.
Keep local staging separate from public release commands.


## Python

```bash
pip install yamlstar-plugin-parser-rapidyaml
```

```python
import yamlstar

opts = yamlstar.Options().plugin(
    yamlstar.parser('rapidyaml'))
ys = yamlstar.YAMLStar(opts)
assert ys.load('a: [1, 2, 3]') == {'a': [1, 2, 3]}
```

Wheels expose the `parser-rapidyaml` entry point in `yamlstar.plugins`.
They contain the ABI v2 shared library and manifest.
Other bindings select the same parser through their existing options mapping.
Integration tests exercise the Glojure CLI and Python binding.

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
Output is owned by the plugin and released with `yamlstar_plugin_v2_free`.
Status 0 returns an EDN event vector.
Status 1 reports parse/configuration errors; status 2 reports ABI/internal errors.
Manifests use EDN; failures return plain UTF-8 error messages.

## Performance

The initial full CLI comparison is slower than built-in `go-yaml` on the
three measured fixtures.
See [Performance.md](Performance.md) for results and reproduction commands.
Those historical measurements used ABI v1 binary transport.
The current ABI v2 adapter converts the internal binary events to EDN.
Assertions are disabled in all builds; parse-error handling remains enabled.

## Release

See [Releasing.md](Releasing.md) for PyPI trusted-publisher setup and
recovery from a failed wheel publication.

```bash
make dist
make wheel
make test-wheel
make release v=0.1.2 d=1
make release v=0.1.2
```

Release archives use
`yamlstar-plugin-parser-rapidyaml-VERSION-PLATFORM.tar.xz`.
The release workflow builds Linux x64/aarch64 with a manylinux 2.28 baseline,
and macOS x64/arm64, validates archives and wheels, and publishes checksums.
The repository must exist under `yamlstar/yamlstar-plugin-parser-rapidyaml`.
Configure PyPI trusted publishing for distribution
`yamlstar-plugin-parser-rapidyaml`, workflow `release.yaml`, environment `pypi`.
Release from `main`, or pass `a=1` to release from another branch.
The release helper creates both `VERSION` and `vVERSION` tags at the same
commit, checks them for conflicts, and updates the adapter's version together
with the native manifest and build metadata.
GitHub assets remain attached to the bare native tag.
The Go tag makes `goyaml` importable at a tagged module version.
Retries preserve both tags; changes after tagging require a new version.
Publish the matching native assets before advertising the adapter release.
No release or publication happens during normal builds or tests.

The integration workflow uses the YAMLStar branch selected by the repository
variable `YAMLSTAR_REF`, defaulting to a pinned main revision with ABI v2
RapidYAML support.
The go-yaml CLI integration uses a pinned host with the canonical parser API.

Third-party notices are included in `licenses/` and release artifacts.
