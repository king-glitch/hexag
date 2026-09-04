// Package parser extracts route metadata from a hexag project's Fiber route
// files using pure go/ast parsing (no go/packages, no reflection) — same
// approach as cmd/mongogen.
package parser

// Route describes one HTTP endpoint discovered under a Register/Register*
// method in a routes/*.go file.
type Route struct {
	Method         string // GET, POST, PUT, PATCH, DELETE, ...
	GroupSegments  []string
	PathSegments   []string // route-local path segments, may include ":id"
	HandlerFile    string   // e.g. "identity.go"
	HandlerType    string   // e.g. "IdentityHandler"
	RegisterMethod string   // "Register", "RegisterAdmin", ...
	HandlerMethod  string   // e.g. "GetMe"
	Doc            string   // handler method's doc comment, if any
	RequiresAuth   bool
	RequestBind    string // "body", "query", "pagination", "none"
	RequestFields  []Field
	ResponseFields []Field
}

// Field describes one example-able field of a request/response struct.
type Field struct {
	Key       string
	Kind      string // string,int,float,bool,objectid,time,object,array,map,collection
	Required  bool
	OneOf     []string
	Email     bool
	SubFields []Field // for object/array-of-object/collection item
}
