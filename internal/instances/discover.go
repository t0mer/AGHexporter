package instances

// DiscoverInstances collects instances from all three input formats (A, B, C)
// and returns the combined, validated, deduplicated list.
//
// instanceFlags is the list of raw --instance flag values from the CLI (Format C).
//
// This is a stub — the real implementation is provided in Task 3.
func DiscoverInstances(instanceFlags []string) ([]Instance, error) {
	panic("not implemented")
}

// ResolvePort applies the documented precedence rule for the --port flag:
//  1. If ADGUARD_EXPORTER_PORT is set in the environment, that value wins.
//  2. Otherwise the supplied flagPort is used.
//
// This is a stub — the real implementation is provided in Task 3.
func ResolvePort(flagPort int) int {
	panic("not implemented")
}
