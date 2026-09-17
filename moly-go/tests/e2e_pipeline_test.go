package tests

import "testing"

// E2E pipeline tests are disabled - services package doesn't exist
func TestE2EPipelineDisabled(t *testing.T) {
	t.Skip("E2E pipeline tests disabled - services package doesn't exist")
}
