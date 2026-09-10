#!/usr/bin/env python3

import hashlib
from importlib import metadata
from pathlib import Path
import sys
import zipfile

import yamlstar_plugin_parser_rapidyaml as plugin


def checksum(path):
  digest = hashlib.sha256()
  with open(path, 'rb') as stream:
    for chunk in iter(lambda: stream.read(1024 * 1024), b''):
      digest.update(chunk)
  return digest.hexdigest()


def plugin_entry_points():
  entry_points = metadata.entry_points()
  if hasattr(entry_points, 'select'):
    return list(entry_points.select(group='yamlstar.plugins'))
  return list(entry_points.get('yamlstar.plugins', ()))


def main():
  wheel = Path(sys.argv[1])
  expected_tag = sys.argv[2]
  source_library = Path(sys.argv[3])
  expected_version = sys.argv[4]

  with zipfile.ZipFile(str(wheel)) as archive:
    names = archive.namelist()
    wheel_metadata = next(
      name for name in names if name.endswith('.dist-info/WHEEL'))
    metadata_text = archive.read(wheel_metadata).decode()
    assert 'Root-Is-Purelib: false' in metadata_text
    assert 'Tag: %s' % expected_tag in metadata_text
    assert any(name.endswith('/plugin.edn') for name in names)
    assert any(name.endswith('/License') for name in names)

  distribution = metadata.distribution('yamlstar-plugin-parser-rapidyaml')
  assert distribution.version == expected_version
  entry_points = [
    item for item in plugin_entry_points()
    if item.name == 'rapidyaml'
  ]
  assert len(entry_points) == 1
  assert entry_points[0].load()() == plugin.library_dir()
  assert Path(plugin.library_path()).is_file()
  assert Path(plugin.manifest_path()).is_file()
  assert checksum(plugin.library_path()) == checksum(source_library)
  pinned = Path(plugin.library_dir()) / 'yamlstar/plugins/rapidyaml' / expected_version
  assert checksum(pinned / Path(plugin.library_path()).name) == checksum(source_library)
  print(plugin.library_dir())


if __name__ == '__main__':
  main()
