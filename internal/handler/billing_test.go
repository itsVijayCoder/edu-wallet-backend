package handler

import "testing"

func TestParentInvoiceStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{name: "empty", input: "", want: "", ok: true},
		{name: "paid", input: "paid", want: "paid", ok: true},
		{name: "pending alias", input: " PENDING ", want: "issued", ok: true},
		{name: "partial alias", input: "partial", want: "partially_paid", ok: true},
		{name: "overdue", input: "overdue", want: "overdue", ok: true},
		{name: "failed", input: "failed", want: "failed", ok: true},
		{name: "invalid", input: "cancelled", want: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parentInvoiceStatus(tt.input)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("parentInvoiceStatus(%q) = (%q, %v), want (%q, %v)", tt.input, got, ok, tt.want, tt.ok)
			}
		})
	}
}
