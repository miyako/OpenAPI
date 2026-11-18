// Entity class functions exposed on /rest/Company({key})/calculateTax
// @openapi.summary Calculate tax for this specific company
// @openapi.param $year Fiscal tax year
exposed Function calculateTax($year : Integer) : Real
	return 42.5
