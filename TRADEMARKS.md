# AVD-SLIM™ Trademark Policy

The avdslim source code is licensed under the [MIT License](LICENSE). That
license covers the code only. It does **not** grant rights to the project's
names or branding.

## The marks

- **AVD-SLIM™** — the product name.
- **avdslim™** — the package, formula, and action name.

Both are owned by Krunal Bhalala ([@kdbhalala](https://github.com/kdbhalala)).

Typing `avdslim` to run the tool, or building the binary under that name for
your own use, is always fine. This policy is about how you name and present
things you publish.

## Official channels

Only these are official:

- Release assets at <https://github.com/kdbhalala/avdslim/releases>
- The Homebrew formula in this repository (`brew tap kdbhalala/avdslim`)
- `install.sh` from this repository
- The GitHub Action `kdbhalala/avdslim`

Release binaries carry build attestations. Verify one with:

```bash
gh attestation verify avdslim_<version>_<os>_<arch>.tar.gz --repo kdbhalala/avdslim
```

Anything else is unofficial, even if it uses the name.

## You may

- Say your product uses, integrates with, or is based on avdslim.
- Refer to the project by name in articles, talks, tutorials, and reviews.
- Redistribute **unmodified** official release binaries under the name
  (package managers, CI caches, container images), with a link back to this
  repository.

## You may not, without written permission

- Publish a fork, modified build, product, service, package, or action under
  the name AVD-SLIM, avdslim, or a confusingly similar name.
- Imply your fork, distribution, or service is official, endorsed by, or
  affiliated with the avdslim project.
- Sell or resell software under these names.

Forks are welcome under MIT. Give them a different name and say they are
based on avdslim.

## Commercial partnerships and reporting misuse

For commercial partnerships, or to report misuse of the names, open an issue at
<https://github.com/kdbhalala/avdslim/issues>.
