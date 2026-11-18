# 4D Integration: 4D.SystemWorker OpenAPI Server

This folder contains the 4D integration component that demonstrates how to supervise and run `4d-openapi serve` as an independent background process via `4D.SystemWorker`.

## How It Works

1. Place the standalone `4d-openapi` native binary in your 4D project under `Resources/bin/4d-openapi` (or in system PATH).
2. Instantiate `cs.OpenAPIServer.new(8088, "127.0.0.1")`.
3. Call `.start()` on 4D startup (or on demand).
4. Call `.openBrowser()` to open the interactive Scalar / Swagger UI documentation at `http://127.0.0.1:8088/docs`.
5. Call `.stop()` on 4D application shutdown (`On Exit Database Method`).

## Example Usage in 4D

```4d
// On Startup Database Method
var $server : cs.OpenAPIServer:=cs.OpenAPIServer.new(8088; "127.0.0.1")
var $res : Object:=$server.start()

If ($res.success)
	$server.openBrowser()
End if
```

```4d
// On Exit Database Method
$server.stop()
```
