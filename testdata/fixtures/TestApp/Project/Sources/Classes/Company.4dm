// DataClass class functions exposed on /rest/Company
// @openapi.summary Get summary of companies
// @openapi.param $detailLevel Level of details (summary, full)
exposed onHTTPGet Function getSummary($detailLevel : Text) : Object
	var $res : Object:=New object
	return $res
