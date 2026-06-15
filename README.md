# ffreis-latex-compiler

<!-- ffreis-badges:start -->
[![CI](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/FelipeFuhr/ffreis-badges/main/badges/ffreis-latex-compiler/ci.json)](https://github.com/FelipeFuhr/ffreis-latex-compiler/actions) [![License](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/FelipeFuhr/ffreis-badges/main/badges/ffreis-latex-compiler/license.json)](https://github.com/FelipeFuhr/ffreis-latex-compiler/blob/main/LICENSE)
<!-- ffreis-badges:end -->

A Go CLI that compiles LaTeX articles into PDF, HTML, and Medium-safe Markdown. It reads articles from a source repository, layers in reusable LaTeX fragments from an optional snippets repository, and drives a per-format toolchain (tectonic, tex4ht/make4ht, pandoc) behind a ports-and-adapters engine. The full toolchain is bundled in a container image, so the host only needs a container runtime to compile. The tool is consumer-agnostic: every consumer (article source, snippets, blog/posts repos) is supplied via flags, never hardcoded.

## What it does

- Loads articles from `<articles-root>/articles/<slug>/`, each described by `main.tex` plus a sidecar `meta.yaml` (a superset of typical blog frontmatter: `title`, `date`, `slug`, `summary`, `tags`, `canonical_url`, `thumbnail`, `available_languages`, `post_slug`). An optional `images/` directory holds figures.
- Adds a snippets repo (`preambles/ classes/ macros/ bib/ figures/`) to the TeX/BibTeX search paths at compile time. Tectonic ignores `TEXINPUTS`, so snippet directories are passed to it as `-Z search-path=<dir>`; TeX Live engines receive the same dirs via `TEXINPUTS`/`BIBINPUTS`.
- Renders each selected article into `dist/<slug>/`, producing `<slug>.pdf`, `<slug>.html`, and an `index.md` + `images/` subtree shaped exactly like a blog repo's `posts/<slug>/` directory. The Markdown step normalizes image links and prepends generated frontmatter so a compiled article promotes into a blog without reshaping metadata.
- Validates article sources, `meta.yaml`, and snippet references, reusing the downstream blog repo's post rules (flags raw HTML such as `<div>`, title over 250 chars, more than 5 tags, missing/mismatched slug, missing images) so a promoted post cannot bounce in CI.
- Promotes a compiled article into a Markdown blog-repo checkout, optionally opening a draft PR. Promotion is always manual: it never auto-merges and refuses to commit onto `main`/`develop`.

## Usage

```
ffreis-latex-compiler <command> [flags]
```

Commands (`build` and `compile` are aliases):

```bash
# Compile article(s) into dist/<slug>/ — single slug or all articles
ffreis-latex-compiler build \
  -articles-root ../articles -snippets-root ../snippets \
  -slug <slug> -out dist -formats pdf,html,md

# Validate sources, meta.yaml, and snippet references (no LaTeX tools needed)
ffreis-latex-compiler validate \
  -articles-root ../articles -snippets-root ../snippets [-slug <slug>]

# Stage a compiled article into a posts-repo checkout, optionally opening a PR
ffreis-latex-compiler promote \
  -articles-root ../articles -out dist -posts-dir ../posts \
  -slug <slug> [-branch <name>] [-open-pr] [-dry-run]

# Report toolchain availability (tectonic, make4ht, pandoc)
ffreis-latex-compiler doctor [-strict]
```

Flag notes:
- `build`: `-articles-root` (default `.`), `-snippets-root` (optional), `-out` (default `dist`), `-slug` (default: all articles), `-formats` (default `pdf,html,md`; valid entries `pdf`, `html`, `md`).
- `promote`: `-slug` and `-posts-dir` are required; `-branch` defaults to `promote/<post-slug>`; `-open-pr` branches, commits, pushes, and opens a draft PR (via `gh`); `-dry-run` reports without writing.
- `doctor`: `-strict` exits non-zero if any tool is missing.

### Container usage

`build` needs the full LaTeX toolchain, so it runs inside the bundled image — the host needs only podman:

```bash
make container-build                 # build the image (containers/Dockerfile.cli)
make build SLUG=<slug> \
  ARTICLES_ROOT=../articles SNIPPETS_ROOT=../snippets FORMATS=pdf,html,md
```

`make build` mounts the articles and snippets roots read-only, the output dir, and a host Tectonic cache, then runs the compiler in the container. `validate`, `promote`, and `doctor` are pure Go (no LaTeX tools): run them natively with `go run`, the installed binary, or the `make validate-articles`, `make promote SLUG=… [OPEN_PR=1] [DRY_RUN=1]`, and `make doctor` targets. `make build-native` runs `build` against host-installed tools.

## Toolchain

Each output format sits behind a `Renderer` interface; adapters shell out via an injectable `Runner` so command and env construction are unit-tested without the binaries. A Noop renderer stands in for disabled formats, and an adapter whose tool is absent fails loudly rather than silently producing nothing.

| Format | Tool | Notes |
|---|---|---|
| PDF | `tectonic` | Auto-downloads CTAN/LaTeX packages on demand; a single static binary. |
| HTML | `make4ht` (tex4ht) | Ships with TeX Live, which Tectonic deliberately omits. |
| Markdown | `pandoc` | `--to=gfm-raw_html` strips raw HTML, keeping the output Medium-safe. |

These three do not coexist in one lightweight package (Tectonic is a single binary; tex4ht needs a TeX Live install), so the full toolchain lives in `containers/Dockerfile.cli` — a TeX Live base image plus tectonic, pandoc, and the Go binary. Markdown is best-effort for prose: Medium renders no LaTeX math or complex floats, which stay high-fidelity only in PDF/HTML.

## Development

Requires Go (see `go.mod`). The only runtime module dependency is `gopkg.in/yaml.v3`.

```bash
make setup            # install lefthook git hooks + verify dev tools
make test             # go test -race -shuffle=on ./...
make coverage-gate    # tests + enforce coverage minimum (COVERAGE_MIN=90)
make quality-gates    # pre-push gate: test + race + coverage + govulncheck
make lint             # golangci-lint
make validate         # go vet + build
make doctor           # report native toolchain availability
go run ./cmd/ffreis-latex-compiler --help
```

## License

AGPL-3.0. See [LICENSE](LICENSE). Copyright (c) 2026 Felipe Fuhr.
