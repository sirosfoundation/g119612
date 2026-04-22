package pipeline

// LoadLoTL is an alias for LoadLoTE. The unified LoadLoTE function
// auto-classifies documents as LoTE or LoTL based on their scheme type.
// See LoadLoTE for full documentation.
var LoadLoTL = LoadLoTE

func init() {
	RegisterFunction("load-lotl", LoadLoTL)
}
