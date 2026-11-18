// Singleton class functions exposed on /rest/$singleton/MySingleton
shared singleton Class constructor()

// @openapi.summary Retrieve global system configuration
exposed onHTTPGet Function getGlobalConfig() : Object
	return New object("version"; "1.0.0")
