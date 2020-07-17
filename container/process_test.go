package container

import (
	"testing"
)

func TestNewParentProcess(t *testing.T) {
	_, pipe := NewParentProcess(&Process{})
	if pipe == nil {
		t.Errorf("%v", pipe)
	}
}
