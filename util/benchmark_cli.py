"""Compare real CLI runs with identical verified results and alternating order."""

import argparse
import json
import os
from pathlib import Path
import statistics
import subprocess
import time

parser = argparse.ArgumentParser()
parser.add_argument('--cli', required=True)
parser.add_argument('--samples', type=int, default=5)
args = parser.parse_args()
env = dict(os.environ, YAMLSTAR_LIBRARY_PATH=str(Path('lib').resolve()))
fixtures = {
  'long-string': json.dumps({'value': 'x' * (240 * 1024)}),
  'small-values': json.dumps([
    {'name': 'entry-%d' % i, 'values': [i, True, 'hello']} for i in range(4000)]),
  'block-yaml': ''.join('key%d:\n  - %d\n  - text\n' % (i, i)
                        for i in range(4000)),
}
results = {}
for name, text in fixtures.items():
  times = {'go-yaml': [], 'rapidyaml': []}
  expected = None
  for sample in range(args.samples + 1):
    names = list(times)
    if sample % 2:
      names.reverse()
    for implementation in names:
      start = time.perf_counter()
      run = subprocess.run(
        [args.cli, '--plugin=yaml-parser=' + implementation, '--no-install'],
        input=text.encode(), stdout=subprocess.PIPE, stderr=subprocess.PIPE,
        check=True, env=env)
      elapsed = time.perf_counter() - start
      value = json.loads(run.stdout)
      if expected is None:
        expected = value
      assert value == expected, (name, implementation)
      if sample:
        times[implementation].append(elapsed)
  medians = {key: statistics.median(value) for key, value in times.items()}
  results[name] = {'bytes': len(text.encode()), 'seconds': times,
                   'median_seconds': medians,
                   'speedup': medians['go-yaml'] / medians['rapidyaml']}
print(json.dumps(results, indent=2))
