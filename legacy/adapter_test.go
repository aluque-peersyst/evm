package legacy_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/legacy"
	"github.com/cosmos/evm/legacy/legacytestutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func ctxAt(height int64) sdk.Context {
	return sdk.Context{}.WithBlockHeight(height)
}

// loadedAdapter builds an adapter and reads its upgrade height once
func loadedAdapter(t *testing.T, v9Height int64) *legacy.LegacyAdapter {
	t.Helper()
	a := legacy.NewLegacyAdapter(nil, legacytestutil.FakeUpgradeKeeper{V9Height: v9Height}, nil)
	require.NoError(t, a.Load(ctxAt(0)))
	return a
}

func TestIsLegacy(t *testing.T) {
	require.Equal(t, "v9.0.0", legacy.V9)

	adapter := loadedAdapter(t, legacytestutil.V9Height)
	require.True(t, adapter.IsLegacy(ctxAt(legacytestutil.V9Height-1)))
	require.False(t, adapter.IsLegacy(ctxAt(legacytestutil.V9Height)))
	require.False(t, adapter.IsLegacy(ctxAt(legacytestutil.V9Height+1)))
}

func TestIsLegacyNeverUpgraded(t *testing.T) {
	adapter := loadedAdapter(t, 0)
	require.False(t, adapter.IsLegacy(ctxAt(legacytestutil.V9Height-1)))
	require.False(t, adapter.IsLegacy(ctxAt(1)))
}

func TestIsLegacyPanicsBeforeLoad(t *testing.T) {
	adapter := legacy.NewLegacyAdapter(nil, legacytestutil.FakeUpgradeKeeper{}, nil)
	require.PanicsWithValue(t, "legacy: IsLegacy called before Load", func() {
		adapter.IsLegacy(ctxAt(1))
	})
}
