// Package pipeline registers all pipeline step functions.
package pipeline

func init() {
	// Register all pipeline steps
	RegisterFunction("load", LoadTSL)
	RegisterFunction("select", SelectCertPool)           // Main name
	RegisterFunction("select-cert-pool", SelectCertPool) // Alternative name for backward compatibility
	RegisterFunction("echo", Echo)
	RegisterFunction("generate", GenerateTSL)
	RegisterFunction("publish", PublishTSL)
	RegisterFunction("log", Log)
	RegisterFunction("set-fetch-options", SetFetchOptions)
}
