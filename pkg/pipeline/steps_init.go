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

	// LoTE (ETSI TS 119 602) steps
	RegisterFunction("load-lote", LoadLoTE)
	RegisterFunction("generate-lote", GenerateLoTE)
	RegisterFunction("publish-lote", PublishLoTE)
	RegisterFunction("increment-lote-sequence", IncrementLoTESequence)
	// convert-to-lote is registered via its own init() in step_convert_lote.go
	// merge-lote is registered via its own init() in step_merge_lote.go
}
