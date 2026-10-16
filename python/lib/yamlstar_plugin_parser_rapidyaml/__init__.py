# Copyright 2026 yaml.org
# MIT License

"""Installed library paths for the YAMLStar RapidYAML plugin."""

import os
from pathlib import Path
import sys


def _library_filename():
  if sys.platform == 'darwin':
    extension = 'dylib'
  elif sys.platform == 'linux' or sys.platform.startswith('freebsd'):
    extension = 'so'
  else:
    raise RuntimeError(
      "Unsupported platform '%s' for yamlstar-plugin-parser-rapidyaml" %
      sys.platform)
  return 'libyamlstar-plugin-parser-rapidyaml.%s' % extension


def library_path():
  """Return the absolute path to the installed plugin library."""
  path = Path(__file__).parent / 'lib' / _library_filename()
  if not path.is_file():
    raise RuntimeError('YAMLStar RapidYAML library not found: %s' % path)
  return str(path.resolve())


def library_dir():
  """Return the absolute directory containing the plugin library."""
  return os.path.dirname(library_path())


def manifest_path():
  """Return the absolute path to the installed plugin manifest."""
  path = Path(__file__).parent / 'plugin.edn'
  if not path.is_file():
    raise RuntimeError('YAMLStar RapidYAML manifest not found: %s' % path)
  return str(path.resolve())
