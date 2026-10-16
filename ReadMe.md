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
YAMLSTAR_LIBRARY_PATH="$PWD/lib" yaml --plugin=parser=rapidyaml --no-install \
  --eval 'a: [1, 2, 3]'
```

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

```bash
make dist
make wheel
make test-wheel
make release o=0.1.0 v=0.1.1 d=1
make release o=0.1.0 v=0.1.1
```

Release archives use
`yamlstar-plugin-parser-rapidyaml-VERSION-PLATFORM.tar.xz`.
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
