package common

import (
	"testing"
)

func TestValidateInstanceType(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{
			name:  "valid G1Standard4T",
			value: G1Standard4T,
			want:  true,
		},
		{
			name:  "valid G1Standard8T",
			value: G1Standard8T,
			want:  true,
		},
		{
			name:  "invalid empty string",
			value: "",
			want:  false,
		},
		{
			name:  "invalid random value",
			value: "invalid-type",
			want:  false,
		},
		{
			name:  "invalid similar value",
			value: "g1-standard-4",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateInstanceType(tt.value); got != tt.want {
				t.Errorf("ValidateInstanceType(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestGetInstanceTypeByValue(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantFound bool
		wantValue string
		wantName  string
	}{
		{
			name:      "finds G1Standard4T",
			value:     G1Standard4T,
			wantFound: true,
			wantValue: G1Standard4T,
			wantName:  "G1Standard4T",
		},
		{
			name:      "finds G1Standard8T",
			value:     G1Standard8T,
			wantFound: true,
			wantValue: G1Standard8T,
			wantName:  "G1Standard8T",
		},
		{
			name:      "returns false for invalid",
			value:     "invalid",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := GetInstanceTypeByValue(tt.value)
			if found != tt.wantFound {
				t.Errorf("GetInstanceTypeByValue(%q) found = %v, want %v", tt.value, found, tt.wantFound)
				return
			}
			if tt.wantFound {
				if got.Value != tt.wantValue {
					t.Errorf("GetInstanceTypeByValue(%q).Value = %v, want %v", tt.value, got.Value, tt.wantValue)
				}
				if got.Name != tt.wantName {
					t.Errorf("GetInstanceTypeByValue(%q).Name = %v, want %v", tt.value, got.Name, tt.wantName)
				}
			}
		})
	}
}

func TestGetDefaultInstanceType(t *testing.T) {
	got := GetDefaultInstanceType()

	if got.Value != G1Standard4T {
		t.Errorf("GetDefaultInstanceType().Value = %v, want %v", got.Value, G1Standard4T)
	}
	if got.Name != "G1Standard4T" {
		t.Errorf("GetDefaultInstanceType().Name = %v, want %v", got.Name, "G1Standard4T")
	}
}

func TestGetAvailableInstanceTypes(t *testing.T) {
	got := GetAvailableInstanceTypes()

	if len(got) != 2 {
		t.Errorf("GetAvailableInstanceTypes() returned %d types, want 2", len(got))
	}

	// Check first is G1Standard4
	if got[0].Value != G1Standard4T {
		t.Errorf("GetAvailableInstanceTypes()[0].Value = %v, want %v", got[0].Value, G1Standard4T)
	}

	// Check second is G1Standard8
	if got[1].Value != G1Standard8T {
		t.Errorf("GetAvailableInstanceTypes()[1].Value = %v, want %v", got[1].Value, G1Standard8T)
	}
}
