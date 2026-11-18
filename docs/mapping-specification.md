# 4D to OpenAPI 3.1 Mapping Specification

This document details how 4D ORDA structures, class functions, types, and custom HTTP handlers map to OpenAPI 3.1.

## 1. ORDA DataClass REST API (Surface A)

For each exposed DataClass (defined in `catalog.4DCatalog` with `exposed="true"`):

### Collection Endpoints (`/rest/{DataClass}`)
- **`GET /rest/{DataClass}`**:
  - Lists entity collections.
  - Query parameters:
    - `$top`: Maximum number of entities to return (integer).
    - `$skip`: Number of entities to skip (integer).
    - `$filter`: Filter string expression (string).
    - `$params`: JSON array string of parameter placeholder values (string).
    - `$orderby`: Sort expression (string).
    - `$attributes`: Attribute list projection (string).
    - `$method`: Entity set method, e.g. `entityset` (string).
    - `$compute`: Aggregate expression, e.g. `salary:sum` (string).
    - `$entityset`: Entity set identifier (string).
  - Response 200: Schema referencing `#/components/schemas/{DataClass}Selection`.
- **`POST /rest/{DataClass}`**:
  - Creates a new entity.
  - Request body: Schema referencing `#/components/schemas/{DataClass}`.
  - Response 200: Created entity representation.

### Entity Endpoints (`/rest/{DataClass}({key})`)
- **`GET /rest/{DataClass}({key})`**:
  - Path parameter `key`: Primary key of entity (string).
  - Response 200: `#/components/schemas/{DataClass}`.
  - Response 404: Not found.
- **`PUT /rest/{DataClass}({key})`**:
  - Updates existing entity.
- **`DELETE /rest/{DataClass}({key})`**:
  - Deletes entity. Returns `{ "ok": true }`.
- **`GET /rest/{DataClass}({key})/{binaryAttribute}`**:
  - Streams binary Blob or Picture content directly (`image/*` or `application/octet-stream`).

### Catalog & Entity Set Endpoints
- **`GET /rest/$catalog`**: Discovers all exposed DataClasses, attributes, types, and methods in DataStore.
- **`GET /rest/$catalog/{DataClass}`**: Discovers metadata definition for a specific DataClass.
- **`GET /rest/{DataClass}/$entityset/{entitySetId}`**: Queries entities in a saved entity set.
- **`DELETE /rest/{DataClass}/$entityset/{entitySetId}`**: Releases a saved entity set.

## 2. Exposed Class Functions (Surface B)

Exposed functions declared with `exposed Function ...` in 4D classes:

| Class Type | File Pattern | OpenAPI Route |
| :--- | :--- | :--- |
| **DataStore** | `DataStore.4dm` | `/rest/$catalog/{function}` |
| **DataClass** | `{DataClass}.4dm` | `/rest/{DataClass}/{function}` |
| **Entity** | `{DataClass}Entity.4dm` | `/rest/{DataClass}({key})/{function}` |
| **EntitySelection** | `{DataClass}Selection.4dm` | `/rest/{DataClass}/{function}` |
| **Singleton** | `{ClassName}.4dm` | `/rest/$singleton/{ClassName}/{function}` |

### HTTP Methods & onHTTPGet
- **POST**: Generated for all exposed class functions. Accepts parameters in JSON request body.
- **GET**: Generated when function has `onHTTPGet` modifier. Accepts parameters via `$params` query string.

## 3. Custom HTTP Handlers (Surface C)

Configured in `HTTPHandlers.json`:
- Literal route patterns (e.g. `/api/v1/webhook`) map directly to OpenAPI paths.
- Regex routes (e.g. `/api/v1/users/([0-9]+)`) map to OpenAPI path templates (`/api/v1/users/{id}`) and preserve the exact regex in `x-4d-regexPattern`.

## 4. Type Mapping Table

| 4D Type | Catalog ID | OpenAPI 3.1 Type | Format |
| :--- | :--- | :--- | :--- |
| Boolean | 1 | `boolean` | - |
| Integer (16/32 bit) | 2, 3 | `integer` | `int32` |
| Integer 64 bits | 4 | `integer` | `int64` |
| Real / Float | 6 | `number` | `double` |
| Date | 7 | `string` | `date` |
| Time | 8 | `integer` | `int64` (ms) |
| Text / Alpha | 9, 0 | `string` | - |
| Picture | 10 | `string` | `binary` |
| Object | 14 | `object` | - |
| Collection | 15 | `array` | - |
| Blob | 16 | `string` | `byte` |
| Nullable | - | `["type", "null"]` | - |
