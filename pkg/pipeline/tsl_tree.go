package pipeline

import (
	"github.com/sirosfoundation/g119612/pkg/etsi119612"
)

// TSLNode represents a node in the TSL tree
// Each node contains a TSL and its child nodes (referenced TSLs)
type TSLNode struct {
	TSL      *etsi119612.TSL // The TSL at this node
	Children []*TSLNode      // Child nodes (referenced TSLs)
}

// TSLTree represents a hierarchical structure of TSLs
// The root is the top-level TSL, with references organized as a tree
type TSLTree struct {
	Root *TSLNode // Root node of the tree
}

// NewTSLTree creates a new TSL tree with a given root TSL
func NewTSLTree(rootTSL *etsi119612.TSL) *TSLTree {
	if rootTSL == nil {
		return &TSLTree{}
	}

	return &TSLTree{
		Root: buildTSLNode(rootTSL),
	}
}

// buildTSLNode recursively builds a TSL node and its children
func buildTSLNode(tsl *etsi119612.TSL) *TSLNode {
	if tsl == nil {
		return nil
	}

	node := &TSLNode{
		TSL:      tsl,
		Children: make([]*TSLNode, 0),
	}

	// Add all referenced TSLs as children
	for _, ref := range tsl.Referenced {
		if childNode := buildTSLNode(ref); childNode != nil {
			node.Children = append(node.Children, childNode)
		}
	}

	return node
}

// Traverse executes a function on each TSL in the tree in pre-order
// (parent first, then children)
func (tree *TSLTree) Traverse(fn func(*etsi119612.TSL)) {
	if tree.Root == nil {
		return
	}

	traverseNode(tree.Root, fn)
}

// traverseNode recursively traverses a node and its children
func traverseNode(node *TSLNode, fn func(*etsi119612.TSL)) {
	if node == nil || node.TSL == nil {
		return
	}

	// Process this node
	fn(node.TSL)

	// Process all children
	for _, child := range node.Children {
		traverseNode(child, fn)
	}
}

// FindBySource finds a TSL in the tree by its source URL
func (tree *TSLTree) FindBySource(source string) *etsi119612.TSL {
	if tree.Root == nil {
		return nil
	}

	var found *etsi119612.TSL
	tree.Traverse(func(tsl *etsi119612.TSL) {
		if tsl.Source == source {
			found = tsl
		}
	})

	return found
}

// Count returns the total number of TSLs in the tree
func (tree *TSLTree) Count() int {
	count := 0
	tree.Traverse(func(_ *etsi119612.TSL) {
		count++
	})
	return count
}

// ItselfOrChild checks if the given TSL is in the tree
// either as the root or as a referenced TSL
func (tree *TSLTree) ItselfOrChild(tsl *etsi119612.TSL) bool {
	if tree.Root == nil || tsl == nil {
		return false
	}

	var found bool
	tree.Traverse(func(t *etsi119612.TSL) {
		if t == tsl {
			found = true
		}
	})

	return found
}

// ToSlice converts the tree to a flat slice of TSLs
func (tree *TSLTree) ToSlice() []*etsi119612.TSL {
	if tree.Root == nil {
		return []*etsi119612.TSL{}
	}

	var result []*etsi119612.TSL
	tree.Traverse(func(tsl *etsi119612.TSL) {
		result = append(result, tsl)
	})

	return result
}

// FromSlice creates a TSL tree from a flat slice of TSLs
// The first TSL in the slice is assumed to be the root
func FromSlice(tsls []*etsi119612.TSL) *TSLTree {
	if len(tsls) == 0 {
		return &TSLTree{}
	}

	// Take the first TSL as the root
	root := tsls[0]
	return NewTSLTree(root)
}

// Depth returns the maximum depth of the tree (root = 0)
// This is useful for understanding how many levels of referenced TSLs are present
func (tree *TSLTree) Depth() int {
	if tree.Root == nil {
		return 0
	}

	return calculateNodeDepth(tree.Root, 0)
}

// calculateNodeDepth recursively calculates the maximum depth from a node
func calculateNodeDepth(node *TSLNode, currentDepth int) int {
	if node == nil {
		return currentDepth
	}

	// If no children, return current depth
	if len(node.Children) == 0 {
		return currentDepth
	}

	// Find the maximum depth among children
	maxChildDepth := currentDepth
	for _, child := range node.Children {
		childDepth := calculateNodeDepth(child, currentDepth+1)
		if childDepth > maxChildDepth {
			maxChildDepth = childDepth
		}
	}

	return maxChildDepth
}
