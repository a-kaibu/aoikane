package assetfs

import (
	"errors"
	"io"
	"io/fs"
	"testing"
)

func TestRegisteredAssets(t *testing.T) {
	Remove("assets/test/file.txt")
	t.Cleanup(func() { Remove("assets/test/file.txt") })

	Put("./assets/test/file.txt", []byte("hello"))
	if got, ok := Lookup("assets/test/file.txt"); !ok || string(got) != "hello" {
		t.Fatalf("Lookup = %q, %v", got, ok)
	}
	if got, err := ReadFile("assets/test/file.txt"); err != nil || string(got) != "hello" {
		t.Fatalf("ReadFile = %q, %v", got, err)
	}
	f, err := Open("assets/test/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got, err := io.ReadAll(f); err != nil || string(got) != "hello" {
		t.Fatalf("Open read = %q, %v", got, err)
	}

	Remove("assets/test/file.txt")
	if _, ok := Lookup("assets/test/file.txt"); ok {
		t.Fatal("asset remains after Remove")
	}
	if !NotExist(fs.ErrNotExist) || NotExist(errors.New("other")) {
		t.Fatal("NotExist misclassified errors")
	}
}
