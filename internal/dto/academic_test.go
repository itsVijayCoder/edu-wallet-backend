package dto

import (
	"encoding/json"
	"testing"
)

func TestGuardianAddressRequestAcceptsFlatAndStructuredAddresses(t *testing.T) {
	tests := []struct {
		name        string
		payload     string
		wantLine1   string
		wantCity    string
		wantCountry string
	}{
		{
			name:      "flat frontend address",
			payload:   `{"name":"Rajesh Kumar","address":"123, Main Street, City"}`,
			wantLine1: "123, Main Street, City",
		},
		{
			name:        "structured legacy address",
			payload:     `{"name":"Rajesh Kumar","address":{"line1":"123 Main Street","city":"Chennai","country":"India"}}`,
			wantLine1:   "123 Main Street",
			wantCity:    "Chennai",
			wantCountry: "India",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var request CreateGuardianRequest
			if err := json.Unmarshal([]byte(tt.payload), &request); err != nil {
				t.Fatalf("unmarshal guardian request: %v", err)
			}
			if request.Address.Line1 != tt.wantLine1 {
				t.Fatalf("line1 = %q, want %q", request.Address.Line1, tt.wantLine1)
			}
			if request.Address.City != tt.wantCity {
				t.Fatalf("city = %q, want %q", request.Address.City, tt.wantCity)
			}
			if request.Address.Country != tt.wantCountry {
				t.Fatalf("country = %q, want %q", request.Address.Country, tt.wantCountry)
			}
		})
	}
}

func TestGuardianAddressRequestRejectsInvalidShape(t *testing.T) {
	var request CreateGuardianRequest
	if err := json.Unmarshal([]byte(`{"name":"Rajesh Kumar","address":42}`), &request); err == nil {
		t.Fatal("expected invalid guardian address to fail")
	}
}
