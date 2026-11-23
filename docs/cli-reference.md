# 4d-openapi CLI Reference

## Synopsis

```bash
4d-openapi <command> [flags]
```

## Exit Codes

- `0`: Success
- `1`: General error
- `2`: Invalid configuration or command arguments
- `3`: Missing or incompatible `tool4d`
- `4`: Dependency resolution failure
- `5`: Metadata extraction failure
- `6`: OpenAPI 3.1 validation failure
- `7`: Incomplete analysis in strict mode (`--strict`)

## Commands

### `generate`
Extracts 4D project metadata and generates OpenAPI 3.1 specification.

```bash
4d-openapi generate --project ./MyApp.4DProject --format yaml --output ./dist/openapi.yaml
```

**Flags**:
- `--project <path>`: Path to `.4DProject` or project directory (default: `.`)
- `--output <path>`: Path to write specification (default: `openapi.yaml`)
- `--format <yaml|json>`: Output format (default: `yaml`)
- `--config <path>`: Path to `openapi.yaml` configuration file
- `--tool4d <path>`: Explicit path to `tool4d` executable
- `--tool4d-version <version>`: Pinned `tool4d` version
- `--offline`: Strictly offline resolution; no remote dependency fetching
- `--no-dependencies`: Skip external component dependency resolution
- `--emit-ir`: Emit intermediate representation JSON file
- `--emit-manifest`: Emit metadata manifest JSON
- `--strict`: Strict mode: fail if any element is unverified or inferred
- `--stdout`: Write specification directly to stdout
- `--watch`: Watch project files and automatically regenerate on change
- `--timeout <seconds>`: Timeout in seconds for tool4d execution (default: 30)

### `serve`
Runs the independent HTTP documentation server.

```bash
4d-openapi serve --file ./dist/openapi.yaml --listen 127.0.0.1:8088
```

**Flags**:
- `--file <path>`: Path to existing OpenAPI specification
- `--project <path>`: 4D project directory (generates spec in-memory if `--file` omitted)
- `--listen <host:port>`: Listen address (default: `127.0.0.1:8088`)
- `--token <token>`: Optional Bearer authorization token

**Endpoints**:
- `/docs`: Interactive Scalar UI API documentation viewer
- `/openapi.yaml`: Raw YAML specification
- `/openapi.json`: Raw JSON specification
- `/health`: Health status check (`{"status": "ok"}`)

### `inspect`
Quick overview of project surface and diagnostics without generating full specification.

```bash
4d-openapi inspect --project ./MyApp.4DProject [--json]
```

### `dependencies`
Manage and verify component dependencies.

```bash
4d-openapi dependencies list [--project <path>]
4d-openapi dependencies verify [--project <path>]
4d-openapi dependencies lock [--project <path>]
```

### `validate`
Validates an OpenAPI 3.1 specification file.

```bash
4d-openapi validate --file ./dist/openapi.yaml
```

### `diff`
Compares two OpenAPI specifications and detects breaking and non-breaking changes.

```bash
4d-openapi diff --old ./v1/openapi.json --new ./v2/openapi.json [--strict]
```

### `export`
Exports client models or API collections (TypeScript, Dart/Flutter, Swift, C#, Kotlin, or Postman).

```bash
# Export TypeScript interfaces
4d-openapi export --project ./MyApp.4DProject --target typescript --output ./dist/api.d.ts

# Export Dart / Flutter models
4d-openapi export --project ./MyApp.4DProject --target dart --output ./dist/models.dart

# Export Swift Codable structs
4d-openapi export --project ./MyApp.4DProject --target swift --output ./dist/Models.swift

# Export C# (.NET) records with System.Text.Json attributes
4d-openapi export --project ./MyApp.4DProject --target csharp --namespace "MyApp.Models" --output ./dist/Models.cs

# Export Kotlin data classes with kotlinx.serialization
4d-openapi export --project ./MyApp.4DProject --target kotlin --namespace "com.example.app.models" --output ./dist/Models.kt

# Export Postman Collection v2.1.0 JSON
4d-openapi export --project ./MyApp.4DProject --target postman --base-url "http://127.0.0.1:8044" --output ./dist/postman_collection.json
```

### `mock`
Runs an embedded mock REST server matching the specification.

```bash
4d-openapi mock --project ./MyApp.4DProject --listen 127.0.0.1:8088
```

### `init`
Initializes a new `openapi.yaml` configuration and `openapi-overlay.yaml` template.

```bash
4d-openapi init [--project ./MyApp.4DProject]
```
