package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestContainerError(t *testing.T) {
	origErr := errors.New("underlying docker connection failed")
	ce := &ContainerError{
		Description:   "failed to setup container environment",
		OriginalError: origErr,
	}

	errMsg := ce.Error()
	if !strings.Contains(errMsg, "underlying docker connection failed") {
		t.Errorf("Expected ContainerError.Error() to contain original error message, got %q", errMsg)
	}

	ceNoOrig := &ContainerError{
		Description: "some descriptive error",
	}
	if ceNoOrig.Error() != "some descriptive error" {
		t.Errorf("Expected error message to be %q, got %q", "some descriptive error", ceNoOrig.Error())
	}
}
