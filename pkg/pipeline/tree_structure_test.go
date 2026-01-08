package pipeline

import (
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/stretchr/testify/assert"
)

func TestTreeStructureBasic(t *testing.T) {
	// Create a basic TSL tree
	rootTSL := &etsi119612.TSL{
		Source: "root.xml",
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslSchemeTerritory: "ROOT",
			},
		},
	}

	childTSL := &etsi119612.TSL{
		Source: "child.xml",
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslSchemeTerritory: "CHILD",
			},
		},
	}

	rootTSL.Referenced = []*etsi119612.TSL{childTSL}

	// Create the tree
	tree := NewTSLTree(rootTSL)

	// Test tree structure
	assert.NotNil(t, tree.Root)
	assert.Equal(t, rootTSL, tree.Root.TSL)
	assert.Equal(t, 1, len(tree.Root.Children))
	assert.Equal(t, childTSL, tree.Root.Children[0].TSL)

	// Test depth calculation
	// NOTE: Depth() method is not implemented yet
	/*
		rootDepth := tree.Root.Depth()
		assert.Equal(t, 2, rootDepth, "Root node should have depth 2 (itself + child)")

		childDepth := tree.Root.Children[0].Depth()
		assert.Equal(t, 1, childDepth, "Child node should have depth 1 (just itself)")
	*/

	// Test tree traversal
	var visited []*etsi119612.TSL
	tree.Traverse(func(tsl *etsi119612.TSL) {
		visited = append(visited, tsl)
	})

	assert.Equal(t, 2, len(visited), "Should visit 2 TSLs")
	assert.Equal(t, rootTSL, visited[0], "Should visit root first")
	assert.Equal(t, childTSL, visited[1], "Should visit child second")

	// Test tree to slice
	slice := tree.ToSlice()
	assert.Equal(t, 2, len(slice), "Slice should contain 2 TSLs")

	// Test tree in context
	ctx := NewContext()
	ctx.EnsureTSLTrees()
	ctx.AddTSLTree(tree)

	trees := ctx.TSLTrees.ToSlice()
	assert.Equal(t, 1, len(trees), "Context should contain 1 tree")

	// Test getting all TSLs from context
	// NOTE: GetAllTSLs() method is not implemented yet
	/*
		allTSLs := ctx.GetAllTSLs()
		assert.Equal(t, 2, len(allTSLs), "Should get 2 TSLs from context")
	*/
}
