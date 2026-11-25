# 4d-openapi Demo Project

This directory contains a complete, runnable 4D project layout demonstrating all core features of `4d-openapi`:

- **Declarative Catalog (`catalog.4DCatalog`)**: Tables (`Company`, `Employee`), attributes with types and primary keys, and ORDA relations (`one-to-many`, `many-to-one`).
- **Class Declarations (`Classes/Company.4dm`)**: Functions exposed to the REST API with `@openapi.summary` annotations and `onHTTPGet` support.
- **Custom Route Handlers (`HTTPHandlers.json`)**: Custom endpoints mapping to 4D methods (`/api/v1/health`, `/api/v1/webhook`).
- **Configuration & Overlays (`openapi.yaml`, `openapi-overlay.yaml`)**: Spec customization with strict validation mode.

---

## Example Commands

Run these commands from the repository root:

### 1. Inspect the Project API Surface
```bash
./4d-openapi inspect --project ./examples/demo-project
```

### 2. Generate OpenAPI 3.1 YAML & JSON Specifications
```bash
# Generate deterministic OpenAPI 3.1 YAML
./4d-openapi generate --project ./examples/demo-project --output ./examples/demo-project/openapi.gen.yaml

# Generate OpenAPI 3.1 JSON and Intermediate Representation (IR)
./4d-openapi generate --project ./examples/demo-project --output ./examples/demo-project/openapi.gen.json --format json --emit-ir
```

### 3. Serve Interactive Documentation (Scalar UI)
```bash
./4d-openapi serve --project ./examples/demo-project --listen 127.0.0.1:8088
```
Open [http://127.0.0.1:8088/docs](http://127.0.0.1:8088/docs) in your browser.

### 4. Run Mock REST Server
```bash
./4d-openapi mock --project ./examples/demo-project --listen 127.0.0.1:8089
```

### 5. Export Typed Client Models & Postman Collection
```bash
# TypeScript
./4d-openapi export --project ./examples/demo-project --target typescript --output ./examples/dist/models.d.ts

# Swift
./4d-openapi export --project ./examples/demo-project --target swift --output ./examples/dist/Models.swift

# Kotlin
./4d-openapi export --project ./examples/demo-project --target kotlin --namespace "com.example.demo" --output ./examples/dist/Models.kt

# C# (.NET)
./4d-openapi export --project ./examples/demo-project --target csharp --namespace "Demo.Models" --output ./examples/dist/Models.cs

# Dart / Flutter
./4d-openapi export --project ./examples/demo-project --target dart --output ./examples/dist/models.dart

# Postman Collection v2.1.0
./4d-openapi export --project ./examples/demo-project --target postman --output ./examples/dist/collection.json
```
