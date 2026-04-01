# qq

A local markdown search engine. Single Go binary with CLI search and a web
frontend for full-text searching and reading markdown files.

## Features

- Full-text search over local markdown files using [Bleve](https://blevesearch.com/)
- Incremental indexing: detects added, removed, and changed files on restart
- Supports both [markdowner](https://github.com/mbrt/markdowner) and Obsidian frontmatter formats
- Web UI with search and reading experience
- CLI search for quick terminal lookups

## Install

```bash
go install github.com/mbrt/qq/cmd/qq@latest
```

## Configuration

Create `~/.config/qq/config.yaml`:

```yaml
directories:
  - path: ~/notes
  - path: ~/articles
index_path: ~/.local/share/qq/index
```

## Usage

### Search from the terminal

```bash
qq search "concurrency in go"
```

### Start the web server

```bash
qq serve --port 8080
```

Then open http://localhost:8080 in your browser.

## Design

See [docs/design.md](docs/design.md) for the full design document.
