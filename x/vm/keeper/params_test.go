package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/legacy"
	"github.com/cosmos/evm/legacy/legacytestutil"
	legacytypes "github.com/cosmos/evm/rpc/types/legacy"
	"github.com/cosmos/evm/x/vm/types"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/testutil"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
)

// newParamsKeeper avoids NewKeeper, whose global chainConfig singleton panics
// on repeated calls.
func newParamsKeeper(t *testing.T) (Keeper, testutil.TestContext) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey("transient_test")
	testCtx := testutil.DefaultContextWithDB(t, key, tkey)
	encCfg := moduletestutil.MakeTestEncodingConfig()
	adapter := legacy.NewLegacyAdapter(encCfg.Codec, legacytestutil.FakeUpgradeKeeper{V9Height: legacytestutil.V9Height}, nil)
	require.NoError(t, adapter.Load(testCtx.Ctx))
	k := Keeper{
		cdc:           encCfg.Codec,
		storeKey:      key,
		legacyAdapter: adapter,
	}
	return k, testCtx
}

func TestGetParamsEmptyStore(t *testing.T) {
	k, testCtx := newParamsKeeper(t)
	params := k.GetParams(testCtx.Ctx)
	require.Equal(t, types.Params{}, params)
}

func TestGetParamsCurrentSchema(t *testing.T) {
	k, testCtx := newParamsKeeper(t)
	ctx := testCtx.Ctx.WithBlockHeight(legacytestutil.V9Height)

	original := types.Params{
		EvmDenom:                "aXRP",
		ExtraEIPs:               []int64{3855},
		EVMChannels:             []string{"channel-0"},
		ActiveStaticPrecompiles: []string{"0x0000000000000000000000000000000000000800"},
		AccessControl: types.AccessControl{
			Create: types.AccessControlType{
				AccessType: types.AccessTypePermissionless,
			},
			Call: types.AccessControlType{
				AccessType: types.AccessTypePermissionless,
			},
		},
	}

	bz, err := k.cdc.Marshal(&original)
	require.NoError(t, err)
	ctx.KVStore(k.storeKey).Set(types.KeyPrefixParams, bz)

	params := k.GetParams(ctx)
	require.Equal(t, original.EvmDenom, params.EvmDenom)
	require.Equal(t, original.ExtraEIPs, params.ExtraEIPs)
	require.Equal(t, original.EVMChannels, params.EVMChannels)
	require.Equal(t, original.ActiveStaticPrecompiles, params.ActiveStaticPrecompiles)
}

func TestGetParamsLegacySchema(t *testing.T) {
	k, testCtx := newParamsKeeper(t)
	ctx := testCtx.Ctx.WithBlockHeight(legacytestutil.V9Height - 1)

	legacyParams := legacytypes.Params{
		EvmDenom:                "aXRP",
		ExtraEIPs:               []string{"ethereum_3855"},
		EVMChannels:             []string{"channel-0"},
		ActiveStaticPrecompiles: []string{"0x0000000000000000000000000000000000000800"},
		AccessControl: legacytypes.AccessControl{
			Create: legacytypes.AccessControlType{
				AccessType: legacytypes.AccessTypePermissionless,
			},
			Call: legacytypes.AccessControlType{
				AccessType: legacytypes.AccessTypePermissionless,
			},
		},
	}

	bz, err := k.cdc.Marshal(&legacyParams)
	require.NoError(t, err)
	ctx.KVStore(k.storeKey).Set(types.KeyPrefixParams, bz)

	params := k.GetParams(ctx)
	require.Equal(t, "aXRP", params.EvmDenom)
	require.Equal(t, []int64{3855}, params.ExtraEIPs)
	require.Equal(t, []string{"channel-0"}, params.EVMChannels)
	require.Equal(t, []string{"0x0000000000000000000000000000000000000800"}, params.ActiveStaticPrecompiles)
	require.Equal(t, uint64(0), params.HistoryServeWindow)
	require.Nil(t, params.ExtendedDenomOptions)
}

// Bytes in the other era's layout must panic instead of decoding as garbage.
func TestGetParamsWrongSchemaPanics(t *testing.T) {
	k, testCtx := newParamsKeeper(t)
	legacyCtx := testCtx.Ctx.WithBlockHeight(legacytestutil.V9Height - 1)
	currentCtx := testCtx.Ctx.WithBlockHeight(legacytestutil.V9Height)

	current := types.Params{EvmDenom: "aXRP", HistoryServeWindow: 8192}
	bz, err := k.cdc.Marshal(&current)
	require.NoError(t, err)
	testCtx.Ctx.KVStore(k.storeKey).Set(types.KeyPrefixParams, bz)
	require.Panics(t, func() { k.GetParams(legacyCtx) })

	legacyParams := legacytypes.Params{
		EvmDenom:                "aXRP",
		ActiveStaticPrecompiles: []string{"0x0000000000000000000000000000000000000800"},
	}
	bz, err = k.cdc.Marshal(&legacyParams)
	require.NoError(t, err)
	testCtx.Ctx.KVStore(k.storeKey).Set(types.KeyPrefixParams, bz)
	require.Panics(t, func() { k.GetParams(currentCtx) })
}

func TestGetParamsNoAdapter(t *testing.T) {
	k, testCtx := newParamsKeeper(t)
	k.legacyAdapter = nil
	ctx := testCtx.Ctx.WithBlockHeight(legacytestutil.V9Height - 1)

	original := types.Params{EvmDenom: "aXRP", HistoryServeWindow: 8192}
	bz, err := k.cdc.Marshal(&original)
	require.NoError(t, err)
	ctx.KVStore(k.storeKey).Set(types.KeyPrefixParams, bz)

	params := k.GetParams(ctx)
	require.Equal(t, original.EvmDenom, params.EvmDenom)
	require.Equal(t, original.HistoryServeWindow, params.HistoryServeWindow)
}

func TestGetParamsGarbagePanics(t *testing.T) {
	k, testCtx := newParamsKeeper(t)
	ctx := testCtx.Ctx.WithBlockHeight(legacytestutil.V9Height)

	ctx.KVStore(k.storeKey).Set(types.KeyPrefixParams, []byte{0xFF, 0xFE, 0xFD})

	require.Panics(t, func() {
		k.GetParams(ctx)
	})
}
