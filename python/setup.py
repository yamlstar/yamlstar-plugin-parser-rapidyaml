#!/usr/bin/env python3

import re
from pathlib import Path

from setuptools import setup
from wheel.bdist_wheel import bdist_wheel


class PlatformWheel(bdist_wheel):
  """Build a Python-independent wheel containing a native library."""

  def finalize_options(self):
    super().finalize_options()
    self.root_is_pure = False

  def get_tag(self):
    _, _, platform = super().get_tag()
    return 'py3', 'none', platform


base_dir = Path(__file__).parent
project_dir = base_dir.parent
manifest = (project_dir / 'plugin.edn').read_text()
match = re.search(r'^ :version "([^"]+)"$', manifest, re.MULTILINE)
if not match:
  raise ValueError('Plugin version is missing from plugin.edn')

setup(
  name='yamlstar-plugin-parser-rapidyaml',
  version=match.group(1),
  description='RapidYAML parser plugin for YAMLStar',
  long_description=(project_dir / 'ReadMe.md').read_text(),
  long_description_content_type='text/markdown',
  author='Ingy döt Net',
  author_email='ingy@ingy.net',
  url='https://github.com/yamlstar/yamlstar-plugin-parser-rapidyaml',
  license='MIT',
  packages=['yamlstar_plugin_parser_rapidyaml'],
  package_dir={'': 'lib'},
  package_data={
    'yamlstar_plugin_parser_rapidyaml': ['License', 'lib/*', 'lib/yamlstar/plugins/rapidyaml/*/*', 'licenses/*', 'plugin.edn'],
  },
  include_package_data=False,
  python_requires='>=3.6, <4',
  entry_points={
    'yamlstar.plugins': [
      'rapidyaml = yamlstar_plugin_parser_rapidyaml:library_dir',
    ],
  },
  cmdclass={'bdist_wheel': PlatformWheel},
  classifiers=[
    'Development Status :: 3 - Alpha',
    'Intended Audience :: Developers',
    'License :: OSI Approved :: MIT License',
    'Programming Language :: Python :: 3',
    'Topic :: Software Development :: Libraries',
  ],
)
