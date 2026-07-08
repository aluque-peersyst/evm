package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/legacy"
	"github.com/cosmos/evm/legacy/legacytestutil"
	legacytypes "github.com/cosmos/evm/rpc/types/legacy/feemarket"
	"github.com/cosmos/evm/x/feemarket/types"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/testutil"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
)

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

func TestGetParamsEmptyStoreReturnsDefaults(t *testing.T) {
	k, testCtx := newParamsKeeper(t)
	params := k.GetParams(testCtx.Ctx)
	require.Equal(t, types.DefaultParams(), params)
}

func TestGetParamsCurrentSchema(t *testing.T) {
	k, testCtx := newParamsKeeper(t)
	ctx := testCtx.Ctx.WithBlockHeight(legacytestutil.V9Height)

	original := types.DefaultParams()
	original.BaseFee = sdkmath.LegacyNewDec(10_000_000_000)

	bz, err := k.cdc.Marshal(&original)
	require.NoError(t, err)
	ctx.KVStore(k.storeKey).Set(types.ParamsKey, bz)

	params := k.GetParams(ctx)
	require.Equal(t, original.BaseFee, params.BaseFee)
	require.Equal(t, original.MinGasPrice, params.MinGasPrice)
}

func TestGetParamsLegacySchema(t *testing.T) {
	k, testCtx := newParamsKeeper(t)
	ctx := testCtx.Ctx.WithBlockHeight(legacytestutil.V9Height - 1)

	legacyParams := legacytypes.Params{
		BaseFeeChangeDenominator: 8,
		ElasticityMultiplier:     2,
		BaseFee:                  sdkmath.NewInt(10_000_000_000),
		MinGasPrice:              sdkmath.LegacyNewDecWithPrec(25, 3),
		MinGasMultiplier:         sdkmath.LegacyNewDecWithPrec(5, 1),
	}

	bz, err := k.cdc.Marshal(&legacyParams)
	require.NoError(t, err)
	ctx.KVStore(k.storeKey).Set(types.ParamsKey, bz)

	params := k.GetParams(ctx)
	require.Equal(t, "10000000000.000000000000000000", params.BaseFee.String())
	require.Equal(t, uint32(8), params.BaseFeeChangeDenominator)
	require.Equal(t, sdkmath.LegacyNewDecWithPrec(25, 3), params.MinGasPrice)
}

// The wire-compatible schemas decode the same bytes to different magnitudes:
// the height dispatch is the only safeguard.
func TestGetParamsSchemaDispatch(t *testing.T) {
	k, testCtx := newParamsKeeper(t)

	legacyParams := legacytypes.Params{
		BaseFee:          sdkmath.NewInt(10_000_000_000),
		MinGasPrice:      sdkmath.LegacyZeroDec(),
		MinGasMultiplier: sdkmath.LegacyZeroDec(),
	}
	bz, err := k.cdc.Marshal(&legacyParams)
	require.NoError(t, err)
	testCtx.Ctx.KVStore(k.storeKey).Set(types.ParamsKey, bz)

	legacyCtx := testCtx.Ctx.WithBlockHeight(legacytestutil.V9Height - 1)
	currentCtx := testCtx.Ctx.WithBlockHeight(legacytestutil.V9Height)

	require.Equal(t, "10000000000.000000000000000000", k.GetParams(legacyCtx).BaseFee.String())
	require.Equal(t, "0.000000010000000000", k.GetParams(currentCtx).BaseFee.String())
}

func TestGetParamsNoAdapter(t *testing.T) {
	k, testCtx := newParamsKeeper(t)
	k.legacyAdapter = nil
	ctx := testCtx.Ctx.WithBlockHeight(legacytestutil.V9Height - 1)

	original := types.DefaultParams()
	bz, err := k.cdc.Marshal(&original)
	require.NoError(t, err)
	ctx.KVStore(k.storeKey).Set(types.ParamsKey, bz)

	params := k.GetParams(ctx)
	require.Equal(t, original.BaseFee, params.BaseFee)
}

func TestGetParamsGarbagePanics(t *testing.T) {
	k, testCtx := newParamsKeeper(t)
	testCtx.Ctx.KVStore(k.storeKey).Set(types.ParamsKey, []byte{0xFF, 0xFE, 0xFD})

	// Current path panics inside cdc.MustUnmarshal; the legacy path panics via
	// the explicit panic(err) after UnmarshalFeeMarketParams fails.
	require.Panics(t, func() { k.GetParams(testCtx.Ctx.WithBlockHeight(legacytestutil.V9Height)) })
	require.Panics(t, func() { k.GetParams(testCtx.Ctx.WithBlockHeight(legacytestutil.V9Height - 1)) })
}
