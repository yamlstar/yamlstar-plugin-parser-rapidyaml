# PyPI publishing setup and recovery

The release workflow builds and tests native archives and Python wheels
before publishing them to GitHub and PyPI.
A successful GitHub release does not imply that PyPI publication succeeded.

## Configure trusted publishing

Register the following GitHub trusted publisher on **PyPI**, not TestPyPI:

| Field | Value |
|-------|-------|
| PyPI project | `yamlstar-plugin-parser-rapidyaml` |
| GitHub owner | `yamlstar` |
| GitHub repository | `yamlstar-plugin-parser-rapidyaml` |
| Workflow filename | `release.yaml` |
| Environment | `pypi` |

For an existing project, open its **Manage > Publishing** page.
For a first publication, create a pending publisher at
<https://pypi.org/manage/account/publishing/> using that project name.
Use the workflow filename alone, without `.github/workflows/`.
The environment must match the `publish-pypi` job in the workflow.
Do not use the older artifact name `yamlstar-plugin-rapidyaml`.

This configuration must be saved by an authorized PyPI account owner.
Repository files and GitHub Actions permissions cannot create this trust.
The workflow already grants `id-token: write` and uses environment `pypi`.
No long-lived PyPI API token is needed.

See the official instructions for
[existing projects](https://docs.pypi.org/trusted-publishers/adding-a-publisher/)
and [first publications](https://docs.pypi.org/trusted-publishers/creating-a-project-through-oidc/).

## Diagnose a failed publication

Release run
[38920046137](https://github.com/yamlstar/yamlstar-plugin-parser-rapidyaml/actions/runs/38920046137)
built and tested all four platform archives and wheels, and uploaded the
GitHub release assets.
Only the PyPI publishing job failed, with `invalid-publisher`.
PyPI accepted the token's validity but found no matching trusted publisher.
The reported repository, workflow, and environment match the table above.

Correct the PyPI publisher configuration before another publication.
This error is not a wheel build or platform-tag failure.
See [PyPI troubleshooting](https://docs.pypi.org/trusted-publishers/troubleshooting/).

After correcting the configuration, an unchanged release can be recovered
by rerunning only the failed job in GitHub Actions.
Do not rerun the entire workflow: verification rejects releases that already
have assets.
The repository's `make release-retry v=VERSION` also reruns failed jobs when
the release already has assets, provided the local version still matches.

## Release the ABI v2 port

Release 0.1.0 contains the old ABI v1 plugin.
The ABI v2 port requires a new version; do not replace the old release tag
or assets with ABI v2 binaries.
After committing changes and configuring PyPI, preview and release with:

```bash
make release o=0.1.0 v=0.1.1 a=1 d=1
make release o=0.1.0 v=0.1.1 a=1
```

The `a=1` option permits release from the current development branch.
The matching YAMLStar host integration must also be available to users.
