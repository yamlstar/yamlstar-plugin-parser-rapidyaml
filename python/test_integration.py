"""Run with PYTHONPATH pointing at the YAMLStar Python binding under test."""

from importlib import metadata
import json
import os
from pathlib import Path
import yamlstar


def main():
  expected = {'a': [1, 2, 3], 'b': 'hello'}
  text = 'a: &items [1, 2, 3]\nb: hello\n'
  version = metadata.version("yamlstar-plugin-parser-rapidyaml")
  instances = [
    yamlstar.YAMLStar(yamlstar.Options().plugin(yamlstar.parser(name)))
    for name in ('go-yaml', 'rapidyaml', 'rapidyaml@' + version)
  ]
  for instance in instances:
    assert instance.load(text) == expected
    assert instance.load('a: &v [1, 2]\nb: *v') == {
      'a': [1, 2], 'b': [1, 2]}
    try:
      instance.load('[unterminated')
    except Exception:
      pass
    else:
      raise AssertionError('accepted invalid YAML')
    assert instance.load('ok: true') == {'ok': True}
  suite = os.environ.get('YAMLSTAR_TEST_SUITE_DIR')
  if suite:
    count = 0
    known = {'6BCT', 'A2M4', 'DC7X', 'J3BT', 'K54U',
             'Y79Y/004', 'Y79Y/006', 'Y79Y/008', 'Y79Y/010'}
    host_root = Path(yamlstar.__file__).resolve().parents[3]
    host_baseline = host_root / 'core/test-suite/expected-fails/load-glojure.yaml'
    known.update(line.split(':', 1)[0] for line in host_baseline.read_text().splitlines()
                 if line and not line.startswith('#'))
    for fixture in sorted(Path(suite).rglob('in.json')):
      case = fixture.parent.relative_to(suite).as_posix()
      if case in known or (fixture.parent / 'error').exists():
        continue
      remaining = fixture.read_text().strip()
      expected = []
      while remaining:
        value, end = json.JSONDecoder().raw_decode(remaining)
        expected.append(value)
        remaining = remaining[end:].strip()
      if not expected:
        continue
      text = (fixture.parent / 'in.yaml').read_bytes().decode()
      actual = instances[1].load_all(text)
      assert actual == expected, (case, actual, expected)
      count += 1
    print('Verified %d YAML test-suite loaded results' % count)
  print('Python parser integration passed, including pinned wheel discovery')


if __name__ == '__main__':
  main()
