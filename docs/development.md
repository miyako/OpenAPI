# 4d-openapi Development Guide & ADRs

## Architectural Decision Records (ADRs)

### ADR-001: Hybrid Extraction Architecture
- **Context**: 4D code cannot be fully parsed by static AST tools without running the 4D engine, but relying solely on runtime reflection risks missing type declarations or failing in headless CI environments without 4D installed.
- **Decision**: Use a hybrid architecture: declarative catalog and HTTP handlers are parsed statically; class functions and annotations are parsed with a dedicated scoped extractor; when `tool4d` is present, runtime introspection verifies and enriches symbols.
- **Status**: Accepted.

### ADR-002: Versioned Intermediate Representation (IR v1)
- **Context**: Decouple 4D extraction logic from OpenAPI 3.1 code generation and future emitters (e.g. client SDK generators or breaking change detectors).
- **Decision**: Define `irVersion: 1` contract validated with JSON Schema (`schemas/ir-v1.schema.json`).
- **Status**: Accepted.

### ADR-003: Deterministic Byte-Identical Output
- **Context**: Avoid phantom Git diffs and CI test flakes.
- **Decision**: All paths, schemas, operations, parameters, and attributes are sorted with a stable comparator. No timestamps are included in the generated spec; timestamps are emitted in a separate manifest via `--emit-manifest`.
- **Status**: Accepted.

### ADR-004: Self-Contained Native Go Implementation
- **Context**: The tool must run with zero runtime external dependencies (no Node.js, Python, or Docker), cross-compile easily, and embed UI assets directly.
- **Decision**: Implement in Go 1.22+ using standard library modules with embedded HTML UI and custom deterministic YAML/JSON serializers.
- **Status**: Accepted.

## Testing & Verification

Run tests:
```bash
go test -v ./...
```

Golden test suite:
- Fixtures located in `testdata/fixtures/TestApp/`
- Golden references in `testdata/golden/`
- Verifies byte-level determinism between multiple compiler runs.
