// Package legacytestutil provides an upgrade-keeper fake shared by the tests
// that exercise LegacyAdapter's height-based schema dispatch.
package legacytestutil

import (
	"context"

	"github.com/cosmos/evm/legacy"
)

// V9Height is the height the v9 migration is treated as applied at in tests.
const V9Height int64 = 100

type FakeUpgradeKeeper struct {
	V9Height int64
}

func (f FakeUpgradeKeeper) GetDoneHeight(_ context.Context, name string) (int64, error) {
	if name == legacy.V9 {
		return f.V9Height, nil
	}
	return 0, nil
}
