// DataClass class functions exposed on /rest/Company
// @openapi.summary Get summary of companies
// @openapi.param $detailLevel Level of details (summary, full)
exposed onHTTPGet Function getSummary($detailLevel : Text) : Object
	var $res : Object:=New object
	return $res

// @openapi.hidden
// Internal helper function that must remain hidden from generated OpenAPI
exposed Function internalHiddenHelper() : Text
	return "secret"
