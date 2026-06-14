# lyrics

Fetch song lyrics and artist/song suggestions from Lyrics.ovh

`lyrics` is a single pure-Go binary. It reads public lyrics data
over plain HTTPS, shapes it into clean records, and prints output that pipes
into the rest of your tools. No API key, nothing to run alongside it.

The same package is also a [resource-URI driver](#use-it-as-a-resource-uri-driver),
so a host program like [ant](https://github.com/tamnd/ant) can address
lyrics as `lyrics://` URIs.

## Install

```bash
go install github.com/tamnd/lyrics-cli/cmd/lyrics@latest
```

Or grab a prebuilt binary from the [releases](https://github.com/tamnd/lyrics-cli/releases), or run
the container image:

```bash
docker run --rm ghcr.io/tamnd/lyrics:latest --help
```

## Usage

```bash
lyrics page <path>                      # fetch one page as a record
lyrics page <path> -o json              # as JSON, ready for jq
lyrics page <path> --template '{{.Body}}'  # just the readable body text
lyrics links <path>                     # the pages it links to, one per line
lyrics --help                           # the whole command tree
```

Every command shares one output contract: `-o table|json|jsonl|csv|tsv|url|raw`,
`--fields` to pick columns, `--template` for a custom line, and `-n` to limit.
The default adapts to where output goes (a table on a terminal, JSONL in a
pipe), so the same command reads well by hand and parses cleanly downstream.

This is a fresh scaffold. It ships one example resource type, `page`, wired end
to end. Model the real lyrics records in `lyrics/` and declare their
operations in `lyrics/domain.go`; each one becomes a command, an HTTP
route, and an MCP tool at once.

## Serve it

The same operations are available over HTTP and as an MCP tool set for agents,
with no extra code:

```bash
lyrics serve --addr :7777    # GET /v1/page/<path>  returns NDJSON
lyrics mcp                   # speak MCP over stdio
```

## Use it as a resource-URI driver

`lyrics` registers a `lyrics` domain the way a program registers a
database driver with `database/sql`. A host enables it with one blank import:

```go
import _ "github.com/tamnd/lyrics-cli/lyrics"
```

Then [ant](https://github.com/tamnd/ant) (or any program that links the package)
dereferences `lyrics://` URIs without knowing anything about lyrics:

```bash
ant get lyrics://page/<path>   # fetch the record
ant cat lyrics://page/<path>   # just the body text
ant ls  lyrics://page/<path>   # the pages it links to, each addressable
ant url lyrics://page/<path>   # the live https URL
```

## Development

```
cmd/lyrics/   thin main: hands cli.NewApp to kit.Run
cli/                 assembles the kit App from the lyrics domain
lyrics/                the library: HTTP client, data models, and domain.go (the driver)
docs/                tago documentation site
```

```bash
make build      # ./bin/lyrics
make test       # go test ./...
make vet        # go vet ./...
```

## Releasing

Push a version tag and GitHub Actions runs GoReleaser, which builds the
archives, Linux packages, the multi-arch GHCR image, checksums, SBOMs, and a
cosign signature:

```bash
git tag v0.1.0
git push --tags
```

The Homebrew and Scoop steps self-disable until their tokens exist, so the first
release works with no extra secrets.

## License

Apache-2.0. See [LICENSE](LICENSE).
