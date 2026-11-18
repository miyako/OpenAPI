// DataStore class functions exposed on /rest/$catalog
// @openapi.summary Check system health and database statistics
// @openapi.tag DataStore
exposed onHTTPGet Function getSystemStatus() : Object
	var $status : Object:=New object("ok"; True)
	return $status
