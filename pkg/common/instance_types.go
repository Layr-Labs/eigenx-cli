package common

// Instance type constants
const (
	// G1Standard4T represents a standard 4-core machine with TEE support
	G1Standard4T = "g1-standard-4t"
	// G1Standard8T represents a standard 8-core machine with TEE support
	G1Standard8T = "g1-standard-8t"

	// EigenMachineTypeEnvVar is the environment variable name for machine type configuration
	EigenMachineTypeEnvVar = "EIGEN_MACHINE_TYPE_PUBLIC"
)

// InstanceType represents a machine instance type configuration
type InstanceType struct {
	Name        string // Display name (e.g., "G1Standard4")
	Value       string // Machine type value (e.g., "g1-standard-4t")
	Description string // Human-readable description
}

// availableInstanceTypes is the canonical list of supported instance types
// Using a package-level variable avoids allocating a new slice on every call
var availableInstanceTypes = []InstanceType{
	{
		Name:        "G1Standard4T",
		Value:       G1Standard4T,
		Description: "4 vCPUs, 16 GB memory (default)",
	},
	{
		Name:        "G1Standard8T",
		Value:       G1Standard8T,
		Description: "8 vCPUs, 32 GB memory",
	},
}

// GetAvailableInstanceTypes returns the list of available instance types
func GetAvailableInstanceTypes() []InstanceType {
	return availableInstanceTypes
}

// GetDefaultInstanceType returns the default instance type (first in the list)
func GetDefaultInstanceType() InstanceType {
	return availableInstanceTypes[0]
}

// ValidateInstanceType checks if the provided instance type value is valid
func ValidateInstanceType(value string) bool {
	_, ok := GetInstanceTypeByValue(value)
	return ok
}

// GetInstanceTypeByValue returns the InstanceType for a given value
func GetInstanceTypeByValue(value string) (InstanceType, bool) {
	for _, it := range availableInstanceTypes {
		if it.Value == value {
			return it, true
		}
	}
	return InstanceType{}, false
}
