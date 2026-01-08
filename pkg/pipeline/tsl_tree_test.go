package pipeline

import (
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
)

func TestTSLTree(t *testing.T) {
	// Create test TSLs with references
	rootTSL := &etsi119612.TSL{Source: "root.xml"}
	refTSL1 := &etsi119612.TSL{Source: "ref1.xml"}
	refTSL2 := &etsi119612.TSL{Source: "ref2.xml"}
	refTSL3 := &etsi119612.TSL{Source: "ref3.xml"}

	// Set up the references
	rootTSL.Referenced = []*etsi119612.TSL{refTSL1, refTSL2}
	refTSL1.Referenced = []*etsi119612.TSL{refTSL3}

	// Create a tree from the root TSL
	tree := NewTSLTree(rootTSL)

	// Test that the tree was built correctly
	if tree.Root.TSL != rootTSL {
		t.Errorf("Root TSL not set correctly")
	}

	// Test child node count
	if len(tree.Root.Children) != 2 {
		t.Errorf("Root should have 2 children, got %d", len(tree.Root.Children))
	}

	// Test traverse function
	var visited []*etsi119612.TSL
	tree.Traverse(func(tsl *etsi119612.TSL) {
		visited = append(visited, tsl)
	})

	// Should have visited 4 TSLs in total
	if len(visited) != 4 {
		t.Errorf("Traverse should visit 4 TSLs, got %d", len(visited))
	}

	// Root TSL should be first in the traversal
	if visited[0] != rootTSL {
		t.Errorf("First TSL visited should be the root")
	}

	// Test finding a TSL by source
	found := tree.FindBySource("ref2.xml")
	if found != refTSL2 {
		t.Errorf("FindBySource failed to find ref2.xml")
	}

	// Test counting TSLs
	count := tree.Count()
	if count != 4 {
		t.Errorf("Count should return 4, got %d", count)
	}

	// Test converting to slice
	slice := tree.ToSlice()
	if len(slice) != 4 {
		t.Errorf("ToSlice should return 4 TSLs, got %d", len(slice))
	}

	// Test empty tree
	emptyTree := &TSLTree{}
	emptyCount := emptyTree.Count()
	if emptyCount != 0 {
		t.Errorf("Empty tree should have count 0, got %d", emptyCount)
	}
}

func TestTSLTreeInContext(t *testing.T) {
	// Create a context
	ctx := NewContext()

	// Ensure TSL trees stack is initialized
	ctx.EnsureTSLTrees()
	if ctx.TSLTrees == nil {
		t.Fatal("TSLTrees should be initialized")
	}

	// Test adding a TSL
	rootTSL := &etsi119612.TSL{Source: "root.xml"}
	refTSL := &etsi119612.TSL{Source: "ref.xml"}
	rootTSL.Referenced = []*etsi119612.TSL{refTSL}

	ctx.AddTSL(rootTSL)

	// Check that the tree was built and added to the stack
	tree, ok := ctx.TSLTrees.Peek()
	if !ok || tree == nil || tree.Root == nil || tree.Root.TSL != rootTSL {
		t.Fatal("TSLTree root was not set correctly")
	}

	// Test that copying preserves the tree
	newCtx := ctx.Copy()
	newTree, ok := newCtx.TSLTrees.Peek()
	if !ok || newTree == nil || newTree.Root == nil || newTree.Root.TSL != rootTSL {
		t.Fatal("TSLTree was not copied correctly")
	}

	// Test traversal in copied context
	var visited []*etsi119612.TSL
	newTree.Traverse(func(tsl *etsi119612.TSL) {
		visited = append(visited, tsl)
	})

	// Should have visited both TSLs
	if len(visited) != 2 {
		t.Errorf("Traverse should visit 2 TSLs, got %d", len(visited))
	}
}

func TestNewTSLTree_EdgeCases(t *testing.T) {
	t.Run("Nil TSL returns empty tree", func(t *testing.T) {
		tree := NewTSLTree(nil)
		if tree == nil {
			t.Fatal("NewTSLTree should never return nil")
		}
		if tree.Root != nil {
			t.Error("Tree with nil TSL should have nil root")
		}
	})

	t.Run("TSL with no references", func(t *testing.T) {
		tsl := &etsi119612.TSL{Source: "single.xml"}
		tree := NewTSLTree(tsl)

		if tree.Root == nil {
			t.Fatal("Root should not be nil")
		}
		if len(tree.Root.Children) != 0 {
			t.Errorf("TSL with no references should have 0 children, got %d", len(tree.Root.Children))
		}
	})

	t.Run("Traverse on nil root does nothing", func(t *testing.T) {
		tree := &TSLTree{} // Empty tree with nil root
		called := false
		tree.Traverse(func(tsl *etsi119612.TSL) {
			called = true
		})

		if called {
			t.Error("Traverse should not call function when root is nil")
		}
	})

	t.Run("FindBySource returns nil when not found", func(t *testing.T) {
		tsl := &etsi119612.TSL{Source: "found.xml"}
		tree := NewTSLTree(tsl)

		found := tree.FindBySource("notfound.xml")
		if found != nil {
			t.Error("FindBySource should return nil for missing source")
		}
	})

	t.Run("FindBySource on empty tree", func(t *testing.T) {
		tree := &TSLTree{} // Empty tree
		found := tree.FindBySource("any.xml")
		if found != nil {
			t.Error("FindBySource should return nil on empty tree")
		}
	})
}

func TestBuildTSLNode_EdgeCases(t *testing.T) {
	t.Run("Nil referenced TSL is skipped", func(t *testing.T) {
		rootTSL := &etsi119612.TSL{Source: "root.xml"}
		validRef := &etsi119612.TSL{Source: "valid.xml"}

		// Mix nil and valid references
		rootTSL.Referenced = []*etsi119612.TSL{nil, validRef, nil}

		node := buildTSLNode(rootTSL)

		if node == nil {
			t.Fatal("buildTSLNode should not return nil for valid TSL")
		}

		// Should only have 1 child (the valid one)
		if len(node.Children) != 1 {
			t.Errorf("Expected 1 child (nil refs should be skipped), got %d", len(node.Children))
		}

		if node.Children[0].TSL != validRef {
			t.Error("Child should be the valid reference")
		}
	})

	t.Run("buildTSLNode with nil TSL returns nil", func(t *testing.T) {
		node := buildTSLNode(nil)
		if node != nil {
			t.Error("buildTSLNode should return nil for nil TSL")
		}
	})
}

func TestFromSlice(t *testing.T) {
	t.Run("Empty slice returns empty tree", func(t *testing.T) {
		tree := FromSlice([]*etsi119612.TSL{})
		if tree == nil {
			t.Fatal("FromSlice should never return nil")
		}
		if tree.Root != nil {
			t.Error("Empty slice should produce tree with nil root")
		}
	})

	t.Run("Nil slice returns empty tree", func(t *testing.T) {
		tree := FromSlice(nil)
		if tree == nil {
			t.Fatal("FromSlice should never return nil")
		}
		if tree.Root != nil {
			t.Error("Nil slice should produce tree with nil root")
		}
	})

	t.Run("Single TSL becomes root", func(t *testing.T) {
		tsl := &etsi119612.TSL{Source: "single.xml"}
		tree := FromSlice([]*etsi119612.TSL{tsl})

		if tree.Root == nil {
			t.Fatal("Root should not be nil")
		}
		if tree.Root.TSL != tsl {
			t.Error("First TSL should become root")
		}
	})

	t.Run("Multiple TSLs uses first as root", func(t *testing.T) {
		tsl1 := &etsi119612.TSL{Source: "first.xml"}
		tsl2 := &etsi119612.TSL{Source: "second.xml"}
		tsl3 := &etsi119612.TSL{Source: "third.xml"}

		tree := FromSlice([]*etsi119612.TSL{tsl1, tsl2, tsl3})

		if tree.Root == nil {
			t.Fatal("Root should not be nil")
		}
		if tree.Root.TSL != tsl1 {
			t.Error("First TSL should become root")
		}
		// Note: FromSlice only uses the first TSL, doesn't create children from the rest
	})
}

func TestItselfOrChild(t *testing.T) {
	t.Run("Finds root TSL", func(t *testing.T) {
		rootTSL := &etsi119612.TSL{Source: "root.xml"}
		tree := NewTSLTree(rootTSL)

		if !tree.ItselfOrChild(rootTSL) {
			t.Error("Should find the root TSL")
		}
	})

	t.Run("Finds child TSL", func(t *testing.T) {
		rootTSL := &etsi119612.TSL{Source: "root.xml"}
		childTSL := &etsi119612.TSL{Source: "child.xml"}
		rootTSL.Referenced = []*etsi119612.TSL{childTSL}

		tree := NewTSLTree(rootTSL)

		if !tree.ItselfOrChild(childTSL) {
			t.Error("Should find the child TSL")
		}
	})

	t.Run("Does not find non-existent TSL", func(t *testing.T) {
		rootTSL := &etsi119612.TSL{Source: "root.xml"}
		otherTSL := &etsi119612.TSL{Source: "other.xml"}
		tree := NewTSLTree(rootTSL)

		if tree.ItselfOrChild(otherTSL) {
			t.Error("Should not find TSL that's not in tree")
		}
	})

	t.Run("Returns false for nil TSL", func(t *testing.T) {
		rootTSL := &etsi119612.TSL{Source: "root.xml"}
		tree := NewTSLTree(rootTSL)

		if tree.ItselfOrChild(nil) {
			t.Error("Should return false for nil TSL")
		}
	})

	t.Run("Returns false for empty tree", func(t *testing.T) {
		tree := &TSLTree{} // Empty tree
		tsl := &etsi119612.TSL{Source: "any.xml"}

		if tree.ItselfOrChild(tsl) {
			t.Error("Should return false for empty tree")
		}
	})
}

func TestDepth(t *testing.T) {
	t.Run("Empty tree has depth 0", func(t *testing.T) {
		tree := &TSLTree{}
		if tree.Depth() != 0 {
			t.Errorf("Empty tree should have depth 0, got %d", tree.Depth())
		}
	})

	t.Run("Single TSL has depth 0", func(t *testing.T) {
		tsl := &etsi119612.TSL{Source: "single.xml"}
		tree := NewTSLTree(tsl)

		if tree.Depth() != 0 {
			t.Errorf("Single TSL should have depth 0, got %d", tree.Depth())
		}
	})

	t.Run("TSL with one level of children has depth 1", func(t *testing.T) {
		rootTSL := &etsi119612.TSL{Source: "root.xml"}
		child1 := &etsi119612.TSL{Source: "child1.xml"}
		child2 := &etsi119612.TSL{Source: "child2.xml"}
		rootTSL.Referenced = []*etsi119612.TSL{child1, child2}

		tree := NewTSLTree(rootTSL)

		if tree.Depth() != 1 {
			t.Errorf("Tree with one level should have depth 1, got %d", tree.Depth())
		}
	})

	t.Run("TSL with two levels has depth 2", func(t *testing.T) {
		rootTSL := &etsi119612.TSL{Source: "root.xml"}
		child := &etsi119612.TSL{Source: "child.xml"}
		grandchild := &etsi119612.TSL{Source: "grandchild.xml"}

		child.Referenced = []*etsi119612.TSL{grandchild}
		rootTSL.Referenced = []*etsi119612.TSL{child}

		tree := NewTSLTree(rootTSL)

		if tree.Depth() != 2 {
			t.Errorf("Tree with two levels should have depth 2, got %d", tree.Depth())
		}
	})

	t.Run("TSL with multiple branches uses maximum depth", func(t *testing.T) {
		rootTSL := &etsi119612.TSL{Source: "root.xml"}

		// Branch 1: shallow (depth 1)
		child1 := &etsi119612.TSL{Source: "child1.xml"}

		// Branch 2: deep (depth 3)
		child2 := &etsi119612.TSL{Source: "child2.xml"}
		grandchild := &etsi119612.TSL{Source: "grandchild.xml"}
		greatgrandchild := &etsi119612.TSL{Source: "greatgrandchild.xml"}

		grandchild.Referenced = []*etsi119612.TSL{greatgrandchild}
		child2.Referenced = []*etsi119612.TSL{grandchild}
		rootTSL.Referenced = []*etsi119612.TSL{child1, child2}

		tree := NewTSLTree(rootTSL)

		// Should be 3 (root -> child2 -> grandchild -> greatgrandchild)
		if tree.Depth() != 3 {
			t.Errorf("Tree should have depth 3 (deepest branch), got %d", tree.Depth())
		}
	})
}

func TestToSlice_EdgeCases(t *testing.T) {
	t.Run("Nil root returns empty slice", func(t *testing.T) {
		tree := &TSLTree{Root: nil}
		slice := tree.ToSlice()

		if slice == nil {
			t.Error("ToSlice should return empty slice, not nil")
		}
		if len(slice) != 0 {
			t.Errorf("ToSlice with nil root should return empty slice, got %d elements", len(slice))
		}
	})

	t.Run("Single node tree", func(t *testing.T) {
		tsl := &etsi119612.TSL{Source: "single.xml"}
		tree := NewTSLTree(tsl)

		slice := tree.ToSlice()

		if len(slice) != 1 {
			t.Errorf("Expected 1 TSL, got %d", len(slice))
		}
		if slice[0].Source != "single.xml" {
			t.Errorf("Expected source 'single.xml', got '%s'", slice[0].Source)
		}
	})

	t.Run("Tree with multiple nodes", func(t *testing.T) {
		root := &etsi119612.TSL{Source: "root.xml"}
		child1 := &etsi119612.TSL{Source: "child1.xml"}
		child2 := &etsi119612.TSL{Source: "child2.xml"}
		root.Referenced = []*etsi119612.TSL{child1, child2}

		tree := NewTSLTree(root)
		slice := tree.ToSlice()

		if len(slice) != 3 {
			t.Errorf("Expected 3 TSLs, got %d", len(slice))
		}

		// Check that all TSLs are present
		sources := make(map[string]bool)
		for _, tsl := range slice {
			sources[tsl.Source] = true
		}
		if !sources["root.xml"] {
			t.Error("Missing root.xml in slice")
		}
		if !sources["child1.xml"] {
			t.Error("Missing child1.xml in slice")
		}
		if !sources["child2.xml"] {
			t.Error("Missing child2.xml in slice")
		}
	})
}

func TestTraverseNode_EdgeCases(t *testing.T) {
	t.Run("Traverse handles nil node", func(t *testing.T) {
		tree := &TSLTree{Root: nil}
		count := 0

		tree.Traverse(func(tsl *etsi119612.TSL) {
			count++
		})

		if count != 0 {
			t.Errorf("Traverse on nil root should not call function, got %d calls", count)
		}
	})

	t.Run("Traverse handles node with nil TSL", func(t *testing.T) {
		// Create a node with nil TSL (edge case)
		tree := &TSLTree{
			Root: &TSLNode{
				TSL:      nil, // Nil TSL in node
				Children: []*TSLNode{},
			},
		}
		count := 0

		tree.Traverse(func(tsl *etsi119612.TSL) {
			count++
		})

		if count != 0 {
			t.Errorf("Traverse on node with nil TSL should not call function, got %d calls", count)
		}
	})
}

func TestCalculateNodeDepth_EdgeCases(t *testing.T) {
	t.Run("Nil node returns current depth", func(t *testing.T) {
		depth := calculateNodeDepth(nil, 5)
		if depth != 5 {
			t.Errorf("calculateNodeDepth with nil node should return current depth (5), got %d", depth)
		}
	})

	t.Run("Node with no children returns current depth", func(t *testing.T) {
		node := &TSLNode{
			TSL:      &etsi119612.TSL{Source: "leaf.xml"},
			Children: []*TSLNode{},
		}
		depth := calculateNodeDepth(node, 2)
		if depth != 2 {
			t.Errorf("Leaf node should return current depth (2), got %d", depth)
		}
	})

	t.Run("Node with nil children array", func(t *testing.T) {
		node := &TSLNode{
			TSL:      &etsi119612.TSL{Source: "leaf.xml"},
			Children: nil,
		}
		depth := calculateNodeDepth(node, 3)
		if depth != 3 {
			t.Errorf("Node with nil children should return current depth (3), got %d", depth)
		}
	})
}
