# Development

## Layout

```
acp-go/
  acp (root package)   protocol model, Agent/Client interfaces, side wrappers
  jsonrpc/             JSON-RPC connection engine and errors
  transport/           byte stream adapters (stdio, subprocess, custom)
  examples/            runnable echo agent and client
  tools/badge/         void-style SVG badge generator
  docs/                this site
```

## Verify

```bash
make fmt vet lint test   # gofumpt, go vet, golangci-lint, race tests
make cover cover-check   # coverage report and 90% gate
make fuzz                # 30s fuzz smoke on union decoders
make badge               # regenerate docs/coverage.svg
```

Directly:

```bash
gofumpt -l -w .
go vet ./...
golangci-lint run ./...
staticcheck ./...
gosec -quiet ./...
go test -race ./...
```

## Conventions

- `// SPDX-License-Identifier: 0BSD` on every Go file.
- Tagged unions: discriminator field, one pointer per variant,
  `TypeOf`, marshal inference, `Raw` preservation. See
  [tagged unions](protocol/unions.md).
- Standard library only; new dependencies need a strong reason.
- Tests must exercise real behavior: wire shapes are asserted as
  golden JSON maps, unions round-trip in both directions, and the
  connection tests run over real `io.Pipe` transports.
- The suite layers: unit and wire tests, race tests (`go test -race`),
  fuzz targets on every union decoder, `testing/quick` property tests,
  adversarial frame tests, chaos transport tests, goroutine leak
  checks, benchmarks, and an e2e suite that builds and drives the real
  echo-agent binary as a subprocess.

## CI

`.github/workflows/ci.yml` runs on push and PR: lint, a three-OS test
matrix, the coverage gate plus badge freshness, a fuzz smoke,
govulncheck, gitleaks, zizmor (workflow audit), CodeQL, and dependency
review. All actions are pinned to commit SHAs.

## Release

Tagging `v*` runs `.github/workflows/release.yml`: tests, an SPDX
SBOM signed with cosign (keyless OIDC), SLSA provenance, and a GitHub
release with the artifacts attached.

## Docs

This site is built with [Zensical](https://zensical.org):

```bash
uv tool install zensical   # or: pip install zensical
zensical serve             # live preview on localhost:8000
zensical build             # static site into site/
```

The `docs.yml` workflow builds and deploys it to GitHub Pages on
changes under `docs/`.
