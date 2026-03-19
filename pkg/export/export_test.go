package export

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestGenerate(t *testing.T) {
	dir := t.TempDir()
	frames := []string{"frame1", "frame2"}
	path, err := Generate("Test Spinner", frames, 100*time.Millisecond, dir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if !strings.HasSuffix(path, ".go") {
		t.Errorf("expected .go extension, got %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "package main") {
		t.Error("generated file should contain package main")
	}
	if !strings.Contains(content, "frame1") {
		t.Error("generated file should contain frame data")
	}
}

func TestGenerateSlugification(t *testing.T) {
	dir := t.TempDir()
	path, err := Generate("My Cool: Spinner", []string{"f"}, 80*time.Millisecond, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(path, "my_cool__spinner") {
		t.Errorf("expected slugified name in path, got %q", path)
	}
}

func TestSnippet(t *testing.T) {
	result := Snippet("Test", []string{"a", "b"}, 100*time.Millisecond)
	if !strings.Contains(result, "package main") {
		t.Error("snippet should contain package main")
	}
	if !strings.Contains(result, "100") {
		t.Error("snippet should contain interval value")
	}
}

func TestGenerateEscaping(t *testing.T) {
	dir := t.TempDir()
	frames := []string{"hello\nworld", "test\"quote", "back\\slash"}
	_, err := Generate("Escape Test", frames, 50*time.Millisecond, dir)
	if err != nil {
		t.Fatalf("Generate with special chars failed: %v", err)
	}
}

func TestGenerateEmptyFrames(t *testing.T) {
	dir := t.TempDir()
	path, err := Generate("Empty", []string{}, 100*time.Millisecond, dir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "package main") {
		t.Error("should still generate valid file")
	}
}
