package utils

import "testing"

// TestStack_Operations tests basic stack operations
func TestStack_Operations(t *testing.T) {
	s := NewStack[string]()

	// Test initial state
	if !s.IsEmpty() {
		t.Error("New stack should be empty")
	}
	if s.Size() != 0 {
		t.Error("New stack should have size 0")
	}

	// Test Push and Size
	s.Push("first")
	if s.Size() != 1 {
		t.Error("Stack should have size 1 after push")
	}
	s.Push("second")
	if s.Size() != 2 {
		t.Error("Stack should have size 2 after second push")
	}

	// Test Peek
	if item, ok := s.Peek(); !ok || item != "second" {
		t.Error("Peek should return top item without removing it")
	}
	if s.Size() != 2 {
		t.Error("Size should not change after peek")
	}

	// Test Pop
	if item, ok := s.Pop(); !ok || item != "second" {
		t.Error("Pop should return and remove top item")
	}
	if s.Size() != 1 {
		t.Error("Size should decrease after pop")
	}
	if item, ok := s.Pop(); !ok || item != "first" {
		t.Error("Pop should return and remove remaining item")
	}
	if !s.IsEmpty() {
		t.Error("Stack should be empty after popping all items")
	}

	// Test Pop on empty stack
	if _, ok := s.Pop(); ok {
		t.Error("Pop on empty stack should return false")
	}

	// Test ToSlice
	s.Push("a")
	s.Push("b")
	s.Push("c")
	slice := s.ToSlice()
	if len(slice) != 3 {
		t.Error("ToSlice should return all items")
	}
	if slice[0] != "a" || slice[1] != "b" || slice[2] != "c" {
		t.Error("ToSlice should return items in correct order")
	}

	// Test Clear
	s.Clear()
	if !s.IsEmpty() {
		t.Error("Stack should be empty after Clear")
	}
}
