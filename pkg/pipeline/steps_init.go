// Package pipeline registers all pipeline step functions.
package pipeline

func init() {
	// Unified steps (auto-detect format based on content/extension/structure)
	RegisterFunction("load", Load)         // Auto-detect TSL (XML) or LoTE (JSON)
	RegisterFunction("publish", Publish)   // Publish all trust lists in context
	RegisterFunction("generate", Generate) // Auto-detect from directory structure
	RegisterFunction("merge", Merge)       // Merge all trust lists of same type

	// Explicit TSL (ETSI TS 119 612) steps
	RegisterFunction("load-tsl", LoadTSL)
	RegisterFunction("load-xml", LoadTSL) // Alias
	RegisterFunction("publish-tsl", PublishTSL)
	RegisterFunction("publish-xml", PublishTSL) // Alias
	RegisterFunction("generate-tsl", GenerateTSL)
	RegisterFunction("merge-tsl", MergeTSLs)

	// Explicit LoTE (ETSI TS 119 602) steps
	RegisterFunction("load-lote", LoadLoTE)
	RegisterFunction("load-json", LoadLoTE) // Alias
	RegisterFunction("publish-lote", PublishLoTE)
	RegisterFunction("publish-json", PublishLoTE) // Alias
	RegisterFunction("generate-lote", GenerateLoTE)
	RegisterFunction("increment-lote-sequence", IncrementLoTESequence)
	// convert-to-lote is registered via its own init() in step_convert_lote.go
	// merge-lote is registered via its own init() in step_merge_lote.go

	// Other steps
	RegisterFunction("select", SelectCertPool)           // Main name
	RegisterFunction("select-cert-pool", SelectCertPool) // Backward compatibility
	RegisterFunction("echo", Echo)
	RegisterFunction("log", Log)
	RegisterFunction("set-fetch-options", SetFetchOptions)
}
