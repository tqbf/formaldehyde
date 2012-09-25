package msp43x

import (
	"fmt"
)
// Exports "Kind" to allow client code to distinguish between errors; values
// are in constants.go
type CpuError struct {
	Kind int
	msg  string
}

func (e *CpuError) Error() string {
	return fmt.Sprintf("cpu error %s: %d", e.msg, e.Kind)
}

// Generate a CPU-specific error
func newError(kind int, msg string) *CpuError {
	return &CpuError{
		Kind: kind,
		msg:  msg,
	}
}
