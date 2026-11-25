// Class: Company
Class extends Entity

// Exposed entity function to get annual summary
exposed Function getSummary() : Object
    var $result : Object
    $result:=New object("company"; This.name; "revenue"; This.revenue)
    return $result

// Exposed function callable via HTTP GET
// @openapi.summary Retrieve active employees for company
exposed onHTTPGet Function getActiveEmployees() : Collection
    var $list : Collection
    $list:=This.employees.query("active = :1"; True)
    return $list
