// EntitySelection class functions exposed on /rest/Company/exportReport
// @openapi.summary Export selected companies to structured report
exposed Function exportReport() : Object
	return New object("count"; This.length)
