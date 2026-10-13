package agents

import (
	"os"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestMain(m *testing.M) {
	// Tests never touch the real keychain (a CI runner has none, a developer machine should not get test entries).
	keyring.MockInit()
	os.Exit(m.Run())
}
