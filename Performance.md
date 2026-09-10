# Initial performance measurements

Measured on Linux x64, Intel Core i7-1365U, September 10, 2026.
The plugin uses Go 1.27.1, rapidyaml v0.16.0, and YEBP v1.
The host is the locally built Glojure YAMLStar CLI on `parser-rapidyaml`.

Five alternating CLI samples per parser, after one warm-up pair, produced
these median wall times:

| Input | Bytes | Built-in go-yaml | RapidYAML plugin |
|-------|------:|----------------:|-----------------:|
| One long string | 245773 | 13.84 ms | 17.31 ms |
| Many small JSON values | 225780 | 277.35 ms | 804.01 ms |
| Block YAML | 105780 | 184.75 ms | 226.97 ms |

All compared outputs were equal.
These include process startup, parsing, plugin transport, construction, and
JSON output.
The current external plugin is slower than the built-in parser on these
fixtures; these measurements do not establish a general rapidyaml speedup.

To reproduce from this repository with a built YAMLStar CLI:

```bash
make shell CMD='python util/benchmark_cli.py --cli /path/to/yaml --samples 5'
```

The isolated Go benchmark includes C++ parsing, conversion, and binary
encoding, but excludes the host and shared-library ABI call.
Three 200 ms benchmark samples measured approximately 0.59-0.67 ms for a
13 KB mapping fixture and 0.98-1.07 ms for a 240 KiB string.
Go allocation statistics exclude native C++ allocations.
Run `make benchmark` for repeated parser measurements.
