// Class: OpenAPIServer
// Manages the lifecycle of 4d-openapi serve as a background 4D.SystemWorker.

property worker : 4D.SystemWorker
property port : Integer
property host : Text
property binPath : Text
property isRunning : Boolean

Class constructor($port : Integer:=8088; $host : Text:="127.0.0.1")
	This.port:=$port
	This.host:=$host
	This.isRunning:=False
	This.binPath:=This._locateBinary()

// Locate 4d-openapi binary in Resources or system path
Function _locateBinary() : Text
	var $bin : 4D.File:=File("/PACKAGE/Resources/bin/4d-openapi")
	If ($bin.exists)
		return $bin.platformPath
	End if 
	
	// Fallback to project root or PATH
	return "4d-openapi"

// Start the documentation server
Function start($specPath : Text:="") : Object
	If (This.isRunning)
		return New object("success"; True; "message"; "Server is already running"; "url"; This.getUrl())
	End if 
	
	var $cmd : Collection:=New collection(This.binPath; "serve"; "--listen"; This.host+":"+String(This.port))
	If ($specPath#"")
		$cmd.push("--file"; $specPath)
	Else 
		$cmd.push("--project"; Folder(fk database folder).platformPath)
	End if 
	
	var $options : Object:=New object
	$options.onResponse:=Formula(This._onResponse($1))
	$options.onError:=Formula(This._onError($1))
	$options.onTerminate:=Formula(This._onTerminate($1))
	
	This.worker:=4D.SystemWorker.new($cmd; $options)
	This.isRunning:=True
	
	// Wait brief moment for bind and verify health
	DELAY PROCESS(Current process; 30) // ~0.5 sec
	var $healthy : Boolean:=This.checkHealth()
	
	return New object("success"; $healthy; "url"; This.getUrl())

// Check health endpoint
Function checkHealth() : Boolean
	var $url : Text:="http://"+This.host+":"+String(This.port)+"/health"
	var $req : 4D.HTTPRequest:=4D.HTTPRequest.new($url)
	var $resp : Object:=$req.wait()
	If ($resp.status=200)
		return True
	End if 
	return False

// Open documentation in default browser
Function openBrowser()
	var $url : Text:=This.getUrl()+"/docs"
	OPEN URL($url)

// Return base URL
Function getUrl() : Text
	return "http://"+This.host+":"+String(This.port)

// Stop the documentation server
Function stop()
	If (This.worker#Null)
		This.worker.terminate()
		This.worker:=Null
	End if 
	This.isRunning:=False

Function _onResponse($worker : 4D.SystemWorker)
	var $data : Text:=$worker.response
	LOG EVENT(Into Windows log or standard out; "[4d-openapi] "+$data)

Function _onError($worker : 4D.SystemWorker)
	var $data : Text:=$worker.error
	LOG EVENT(Into Windows log or standard out; "[4d-openapi ERROR] "+$data)

Function _onTerminate($worker : 4D.SystemWorker)
	This.isRunning:=False
	LOG EVENT(Into Windows log or standard out; "[4d-openapi] Server process terminated.")
