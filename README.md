# 4d-openapi

Native, cross-platform OpenAPI 3.1 specification generator and interactive documentation server for 4D projects.

[![Go Report Card](https://goreportcard.com/badge/github.com/miyako/OpenAPI)](https://goreportcard.com/report/github.com/miyako/OpenAPI)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## Key Features

- **Hybrid Extraction**: Combines authoritative 4D project metadata (`catalog.4DCatalog`, `HTTPHandlers.json`), scoped class declaration analysis, and `tool4d` runtime introspection without requiring a full 4D AST parser.
- **Full ORDA Coverage**: Automatically generates endpoints for exposed DataClasses (`/rest/{DataClass}` and `/rest/{DataClass}({key})`), CRUD operations, attributes, and ORDA query parameters (`$top`, `$skip`, `$filter`, `$orderby`, etc.).
- **Exposed Class Functions**: Discovers class functions across all 5 target surfaces:
  - `DataStore` (`/rest/$catalog/{function}`)
  - `DataClass` (`/rest/{DataClass}/{function}`)
  - `Entity` (`/rest/{DataClass}({key})/{function}`)
  - `EntitySelection` (`/rest/{DataClass}/{function}`)
  - `Singleton` (`/rest/$singleton/{class}/{function}`)
  - Full support for `onHTTPGet` GET queries and POST JSON bodies.
- **Custom HTTP Handlers**: Maps literal routes and regex route patterns from `HTTPHandlers.json` (preserving regex patterns via `x-4d-regexPattern`).
- **Intermediate Representation (IR v1)**: Clear, decoupled, versioned JSON data contract validated by JSON Schema (`schemas/ir-v1.schema.json`).
- **Independent Documentation Server**: Embedded HTTP server running modern Scalar UI at `/docs`, with `/openapi.yaml`, `/openapi.json`, and `/health` endpoints. Runs independently of the 4D Web Server.
- **4D Integration**: Ready-to-use 4D class (`OpenAPIServer.4dm`) to launch and supervise the server via `4D.SystemWorker`.
- **Deterministic Output**: Byte-identical output on repeated runs with stable key sorting and manifest support.
- **Zero External Runtime Dependencies**: Single native standalone executable for macOS, Windows, and Linux.

---

## Installation & Build

Build from source with Go 1.22+:

```bash
git clone https://github.com/miyako/OpenAPI.git
cd OpenAPI
go build -o 4d-openapi ./cmd/4d-openapi
```

---

## Quick Start

### 1. Inspect Project API Surface
```bash
./4d-openapi inspect --project /path/to/MyApp.4DProject
```

### 2. Generate OpenAPI 3.1 Specification
```bash
# Generate deterministic YAML
./4d-openapi generate --project /path/to/MyApp.4DProject --output ./dist/openapi.yaml --format yaml

# Generate JSON and emit Intermediate Representation (IR)
./4d-openapi generate --project /path/to/MyApp.4DProject --output ./dist/openapi.json --format json --emit-ir
```

### 3. Serve Interactive Documentation
```bash
./4d-openapi serve --file ./dist/openapi.yaml --listen 127.0.0.1:8088
```
Open [http://127.0.0.1:8088/docs](http://127.0.0.1:8088/docs) in your browser to view the interactive Scalar UI.

---

## Command Reference

| Command | Description |
| :--- | :--- |
| `generate` | Extract 4D metadata and generate OpenAPI 3.1 YAML/JSON (with `--watch`) |
| `serve` | Run embedded HTTP server with interactive Scalar UI docs |
| `inspect` | Summarize DataClasses, functions, handlers, and diagnostics |
| `dependencies` | Manage and lock component dependencies (`list`, `verify`, `lock`) |
| `validate` | Validate an OpenAPI 3.1 specification file |
| `diff` | Compare two OpenAPI specifications and detect breaking changes |
| `export` | Export client models (`--target typescript \| dart \| swift`) |
| `init` | Initialize starter `openapi.yaml` and `openapi-overlay.yaml` |
| `mock` | Run embedded mock 4D REST server matching the specification |

For detailed flags and exit codes, see [CLI Reference](file:///Users/miyako/Documents/GitHub/OpenAPI/docs/cli-reference.md).

---

## Architecture & Mapping

- [Architecture Overview](file:///Users/miyako/Documents/GitHub/OpenAPI/docs/architecture.md)
- [4D-to-OpenAPI Mapping Specification](file:///Users/miyako/Documents/GitHub/OpenAPI/docs/mapping-specification.md)
- [4D Integration Guide](file:///Users/miyako/Documents/GitHub/OpenAPI/4d-integration/README.md)
- [Development Guide & ADRs](file:///Users/miyako/Documents/GitHub/OpenAPI/docs/development.md)

---

## License

MIT License. See [LICENSE](file:///Users/miyako/Documents/GitHub/OpenAPI/LICENSE) for details.
