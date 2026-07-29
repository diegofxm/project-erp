package nit_test

import (
	"errors"
	"testing"

	"github.com/diegofxm/erp/internal/shared/nit"
)

func TestComputeCheckDigit_RealNIT(t *testing.T) {
	got, err := nit.ComputeCheckDigit("6382356")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "7" {
		t.Errorf("got %q, want %q", got, "7")
	}
}

func TestComputeCheckDigit_FormattedInput(t *testing.T) {
	got, err := nit.ComputeCheckDigit("6.382.356")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "7" {
		t.Errorf("got %q, want %q", got, "7")
	}
}

func TestComputeCheckDigit_RemainderBranches(t *testing.T) {
	got, err := nit.ComputeCheckDigit("0")
	if err != nil {
		t.Fatalf("unexpected error for '0': %v", err)
	}
	if got != "0" {
		t.Errorf("'0': got %q, want %q", got, "0")
	}

	got, err = nit.ComputeCheckDigit("4")
	if err != nil {
		t.Fatalf("unexpected error for '4': %v", err)
	}
	if got != "1" {
		t.Errorf("'4': got %q, want %q", got, "1")
	}
}

func TestComputeCheckDigit_Invalid(t *testing.T) {
	inputs := []string{"", "abc", "900-ABC", "1234567890123456"}
	for _, in := range inputs {
		got, err := nit.ComputeCheckDigit(in)
		if !errors.Is(err, nit.ErrInvalidNumber) {
			t.Errorf("input %q: want ErrInvalidNumber, got err=%v val=%q", in, err, got)
		}
	}
}
