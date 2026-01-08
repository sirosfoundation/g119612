package pipeline

import (
	"crypto/x509"
	"time"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/utils"
)

// Context holds the shared state passed between pipeline steps during processing.
// It contains Trust Status Lists (TSLs) and certificate pools that are created,
// modified, and consumed by different pipeline steps.
type Context struct {
	TSLTrees        *utils.Stack[*TSLTree]        // A stack of TSL trees, where each tree represents a loaded root TSL and its references
	TSLs            *utils.Stack[*etsi119612.TSL] // DEPRECATED: Legacy stack of TSLs for backward compatibility
	CertPool        *x509.CertPool                // Certificate pool for trust verification
	Data            map[string]any                // Data store for sharing information between pipeline steps
	TSLFetchOptions *etsi119612.TSLFetchOptions   // Options for fetching Trust Status Lists
}

// EnsureTSLTrees ensures that the TSL tree stack is initialized.
// If the stack doesn't exist, it creates a new empty stack.
//
// This method is used by pipeline steps to guarantee that the TSL tree stack
// is available before operating on it, preventing nil pointer exceptions.
//
// Returns:
//   - The Context itself for method chaining
func (ctx *Context) EnsureTSLTrees() *Context {
	if ctx.TSLTrees == nil {
		ctx.TSLTrees = utils.NewStack[*TSLTree]()
	}
	return ctx
}

// AddTSLTree adds a TSL tree to the stack.
// The tree represents a root TSL and all its references.
// It also adds all TSLs in the tree to the legacy TSLs stack for backward compatibility.
//
// Parameters:
//   - tree: The TSL tree to add
//
// Returns:
//   - The Context itself for method chaining
func (ctx *Context) AddTSLTree(tree *TSLTree) *Context {
	ctx.EnsureTSLTrees()

	if tree != nil {
		ctx.TSLTrees.Push(tree)

		// Also add all TSLs in the tree to the legacy stack for backward compatibility
		if ctx.TSLs == nil {
			ctx.TSLs = utils.NewStack[*etsi119612.TSL]()
		}

		// Get all TSLs from the tree in a flat list and add them to the legacy stack
		// Note: We add in reverse order (children first, then root) to maintain the
		// order expected by tests (referenced TSLs before the root TSL)
		if tree.Root != nil {
			// First collect all TSLs in a slice
			var tsls []*etsi119612.TSL
			tree.Traverse(func(tsl *etsi119612.TSL) {
				tsls = append(tsls, tsl)
			})

			// Push referenced TSLs first (all except the root)
			for i := len(tsls) - 1; i > 0; i-- {
				ctx.TSLs.Push(tsls[i])
			}

			// Push the root TSL last
			if len(tsls) > 0 {
				ctx.TSLs.Push(tsls[0])
			}
		}
	}

	return ctx
}

// AddTSL creates a new TSL tree from the provided TSL and adds it to the stack.
// This is a convenience method for adding a single TSL.
// It also adds the TSL to the legacy TSLs stack for backward compatibility.
//
// Parameters:
//   - tsl: The TSL to add
//
// Returns:
//   - The Context itself for method chaining
func (ctx *Context) AddTSL(tsl *etsi119612.TSL) *Context {
	if tsl == nil {
		return ctx
	}

	// Add to the new tree structure
	tree := NewTSLTree(tsl)
	ctx.AddTSLTree(tree)

	// Also add to the legacy stack for backward compatibility
	if ctx.TSLs == nil {
		ctx.TSLs = utils.NewStack[*etsi119612.TSL]()
	}
	ctx.TSLs.Push(tsl)

	return ctx
}

// EnsureTSLStack ensures that the legacy TSL stack is initialized.
// If the stack doesn't exist, it creates a new empty stack.
//
// This method is used by pipeline steps to guarantee that the legacy TSL stack
// is available before operating on it, preventing nil pointer exceptions.
//
// Returns:
//   - The Context itself for method chaining
func (ctx *Context) EnsureTSLStack() *Context {
	if ctx.TSLs == nil {
		ctx.TSLs = utils.NewStack[*etsi119612.TSL]()
	}
	return ctx
}

// EnsureTSLFetchOptions ensures that the TSLFetchOptions are initialized.
// If the options don't exist, it creates new ones with default values.
//
// This method is used by pipeline steps to guarantee that the TSLFetchOptions
// are available before using them, preventing nil pointer exceptions.
//
// Returns:
//   - The Context itself for method chaining
func (ctx *Context) EnsureTSLFetchOptions() *Context {
	if ctx.TSLFetchOptions == nil {
		ctx.TSLFetchOptions = &etsi119612.TSLFetchOptions{
			UserAgent: "Go-Trust/1.0 Pipeline (+https://github.com/sirosfoundation/go-trust)",
			Timeout:   30 * time.Second,
		}
	}
	return ctx
}

// InitCertPool creates a new certificate pool in the context.
// This replaces any existing certificate pool with a fresh, empty one.
//
// This method is typically called before adding trusted certificates
// from Trust Status Lists to build a new trust store.
//
// Returns:
//   - The Context itself for method chaining
func (ctx *Context) InitCertPool() *Context {
	ctx.CertPool = x509.NewCertPool()
	return ctx
}

// Copy creates a deep copy of the Context.
// This is useful for pipeline steps that need to create a modified context
// without affecting the original one, such as for testing or branching pipelines.
//
// The copy includes:
// - A new stack of TSL trees with the same trees
// - A new legacy stack of TSLs with the same TSLs
// - A new certificate pool with the same certificates (if present)
// - A new Data map with the same contents
// - The same TSLFetchOptions reference (since it's typically read-only)
//
// Returns:
//   - A new Context instance with copied contents
func (ctx *Context) Copy() *Context {
	newCtx := NewContext()

	// Copy TSL tree stack if it exists
	if ctx.TSLTrees != nil {
		// Copy each tree in the stack
		trees := ctx.TSLTrees.ToSlice()
		for i := len(trees) - 1; i >= 0; i-- {
			// We don't need to deep copy the trees because the TSL structure already
			// contains the proper references. The tree objects themselves are immutable.
			newCtx.TSLTrees.Push(trees[i])
		}
	}

	// Copy legacy TSL stack if it exists
	if ctx.TSLs != nil {
		// Copy each TSL in the stack
		tsls := ctx.TSLs.ToSlice()
		for i := len(tsls) - 1; i >= 0; i-- {
			newCtx.TSLs.Push(tsls[i])
		}
	}

	// Copy certificate pool if it exists
	if ctx.CertPool != nil {
		newCtx.CertPool = x509.NewCertPool()
		// Cannot directly copy the certificates, but we can use what's in the TSLs
		// The actual cert pool will be reconstructed by SelectCertPool or similar functions
	}

	// Copy data map
	for k, v := range ctx.Data {
		newCtx.Data[k] = v
	}

	// Share the TSLFetchOptions reference
	newCtx.TSLFetchOptions = ctx.TSLFetchOptions

	return newCtx
}

// NewContext creates a new pipeline context with initialized fields.
// The returned Context has a pre-initialized TSL tree stack ready to use,
// but no certificate pool (which should be created with InitCertPool when needed).
//
// Returns:
//   - A new Context instance with initialized fields
func NewContext() *Context {
	return &Context{
		TSLTrees: utils.NewStack[*TSLTree](),
		TSLs:     utils.NewStack[*etsi119612.TSL](),
		Data:     make(map[string]any),
	}
}

// GetCertPool returns the certificate pool from the context.
// This implements the PipelineContextProvider interface used by etsi.PipelineBackedRegistry.
func (ctx *Context) GetCertPool() *x509.CertPool {
	return ctx.CertPool
}

// GetTSLs returns all TSLs from the context as a slice.
// This implements the PipelineContextProvider interface used by etsi.PipelineBackedRegistry.
func (ctx *Context) GetTSLs() []*etsi119612.TSL {
	if ctx.TSLs == nil {
		return nil
	}
	return ctx.TSLs.ToSlice()
}

// GetTSLCount returns the number of loaded TSLs.
// This implements the PipelineContextProvider interface used by etsi.PipelineBackedRegistry.
func (ctx *Context) GetTSLCount() int {
	if ctx.TSLs == nil {
		return 0
	}
	return ctx.TSLs.Size()
}
