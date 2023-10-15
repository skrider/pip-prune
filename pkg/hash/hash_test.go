package hash

import (
	"testing"
)

func TestHashObject(t *testing.T) {
	data := struct {
		Foo string
	}{
		Foo: "bar",
	}
	hash, err := Hash(data)
	if err != nil {
		t.Errorf("error creating hash: %s", err)
	}
	t.Logf("Successfully created hash: %s", hash)
}
