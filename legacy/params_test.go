package legacy_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/legacy"
	evmlegacytypes "github.com/cosmos/evm/rpc/types/legacy"
	fmlegacytypes "github.com/cosmos/evm/rpc/types/legacy/feemarket"
	vmtypes "github.com/cosmos/evm/x/vm/types"

	sdkmath "cosmossdk.io/math"

	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
)

func newParamsAdapter() (*legacy.LegacyAdapter, moduletestutil.TestEncodingConfig) {
	encCfg := moduletestutil.MakeTestEncodingConfig()
	return legacy.NewLegacyAdapter(encCfg.Codec, nil, nil), encCfg
}

func TestUnmarshalLegacyParams(t *testing.T) {
	adapter, encCfg := newParamsAdapter()

	legacyParams := evmlegacytypes.Params{
		EvmDenom:  "aXRP",
		ExtraEIPs: []string{"ethereum_3855", "ethereum_3860"},
		ChainConfig: evmlegacytypes.ChainConfig{
			DAOForkSupport: true,
		},
		AllowUnprotectedTxs:     false,
		EVMChannels:             []string{"channel-0", "channel-1"},
		ActiveStaticPrecompiles: []string{"0x0000000000000000000000000000000000000800"},
		AccessControl: evmlegacytypes.AccessControl{
			Create: evmlegacytypes.AccessControlType{
				AccessType:        evmlegacytypes.AccessTypePermissioned,
				AccessControlList: []string{"0x1234567890abcdef1234567890abcdef12345678"},
			},
			Call: evmlegacytypes.AccessControlType{
				AccessType: evmlegacytypes.AccessTypePermissionless,
			},
		},
	}

	bz, err := encCfg.Codec.Marshal(&legacyParams)
	require.NoError(t, err)

	result, err := adapter.UnmarshalLegacyParams(bz)
	require.NoError(t, err)

	require.Equal(t, "aXRP", result.EvmDenom)
	require.Equal(t, []int64{3855, 3860}, result.ExtraEIPs)
	require.Equal(t, []string{"channel-0", "channel-1"}, result.EVMChannels)
	require.Equal(t, []string{"0x0000000000000000000000000000000000000800"}, result.ActiveStaticPrecompiles)

	require.Equal(t, vmtypes.AccessTypePermissioned, result.AccessControl.Create.AccessType)
	require.Equal(t, []string{"0x1234567890abcdef1234567890abcdef12345678"}, result.AccessControl.Create.AccessControlList)
	require.Equal(t, vmtypes.AccessTypePermissionless, result.AccessControl.Call.AccessType)

	require.Equal(t, uint64(0), result.HistoryServeWindow)
	require.Nil(t, result.ExtendedDenomOptions)
}

func TestUnmarshalLegacyParamsMinimalParams(t *testing.T) {
	adapter, encCfg := newParamsAdapter()

	legacyParams := evmlegacytypes.Params{
		EvmDenom:                "aXRP",
		ActiveStaticPrecompiles: []string{"0x0000000000000000000000000000000000000800"},
	}

	bz, err := encCfg.Codec.Marshal(&legacyParams)
	require.NoError(t, err)

	result, err := adapter.UnmarshalLegacyParams(bz)
	require.NoError(t, err)

	require.Equal(t, "aXRP", result.EvmDenom)
	require.Empty(t, result.ExtraEIPs)
	require.Empty(t, result.EVMChannels)
	require.Equal(t, []string{"0x0000000000000000000000000000000000000800"}, result.ActiveStaticPrecompiles)
}

func TestUnmarshalLegacyParamsInvalidExtraEIP(t *testing.T) {
	adapter, encCfg := newParamsAdapter()

	legacyParams := evmlegacytypes.Params{
		EvmDenom:                "aXRP",
		ExtraEIPs:               []string{"ethereum_jpc"},
		ActiveStaticPrecompiles: []string{"0x0000000000000000000000000000000000000800"},
	}

	bz, err := encCfg.Codec.Marshal(&legacyParams)
	require.NoError(t, err)

	_, err = adapter.UnmarshalLegacyParams(bz)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid legacy ExtraEIP")
}

func TestUnmarshalLegacyParamsRejectsCurrentSchema(t *testing.T) {
	adapter, encCfg := newParamsAdapter()

	// HistoryServeWindow (field 10, varint) collides with the legacy
	// repeated-string field: the legacy decoder must reject current bytes.
	current := vmtypes.Params{
		EvmDenom:                "aXRP",
		ExtraEIPs:               []int64{3855, 3860},
		EVMChannels:             []string{"channel-0"},
		ActiveStaticPrecompiles: []string{"0x0000000000000000000000000000000000000800"},
		HistoryServeWindow:      8192,
		ExtendedDenomOptions:    &vmtypes.ExtendedDenomOptions{ExtendedDenom: "aXRP"},
	}

	bz, err := encCfg.Codec.Marshal(&current)
	require.NoError(t, err)

	_, err = adapter.UnmarshalLegacyParams(bz)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unmarshal legacy EVM params")
}

func TestUnmarshalLegacyParamsInvalidBytes(t *testing.T) {
	adapter, _ := newParamsAdapter()

	_, err := adapter.UnmarshalLegacyParams([]byte{0xFF, 0xFE, 0xFD})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unmarshal legacy EVM params")
}

func TestUnmarshalFeeMarketParams(t *testing.T) {
	adapter, encCfg := newParamsAdapter()

	baseFee := sdkmath.NewInt(10_000_000_000)
	legacyParams := fmlegacytypes.Params{
		NoBaseFee:                false,
		BaseFeeChangeDenominator: 8,
		ElasticityMultiplier:     2,
		EnableHeight:             123,
		BaseFee:                  baseFee,
		MinGasPrice:              sdkmath.LegacyNewDecWithPrec(25, 3),
		MinGasMultiplier:         sdkmath.LegacyNewDecWithPrec(5, 1),
	}

	bz, err := encCfg.Codec.Marshal(&legacyParams)
	require.NoError(t, err)

	result, err := adapter.UnmarshalFeeMarketParams(bz)
	require.NoError(t, err)

	// base_fee recovered at full magnitude, not scaled by 10^-18.
	require.Equal(t, sdkmath.LegacyNewDecFromInt(baseFee), result.BaseFee)
	require.Equal(t, "10000000000.000000000000000000", result.BaseFee.String())

	require.False(t, result.NoBaseFee)
	require.Equal(t, uint32(8), result.BaseFeeChangeDenominator)
	require.Equal(t, uint32(2), result.ElasticityMultiplier)
	require.Equal(t, int64(123), result.EnableHeight)
	require.Equal(t, sdkmath.LegacyNewDecWithPrec(25, 3), result.MinGasPrice)
	require.Equal(t, sdkmath.LegacyNewDecWithPrec(5, 1), result.MinGasMultiplier)
}

// The happy-path test only covers NoBaseFee=false, so a mis-mapped bool would
// pass unnoticed; pin the true case.
func TestUnmarshalFeeMarketParamsNoBaseFee(t *testing.T) {
	adapter, encCfg := newParamsAdapter()

	legacyParams := fmlegacytypes.Params{
		NoBaseFee:        true,
		BaseFee:          sdkmath.NewInt(10_000_000_000),
		MinGasPrice:      sdkmath.LegacyZeroDec(),
		MinGasMultiplier: sdkmath.LegacyZeroDec(),
	}

	bz, err := encCfg.Codec.Marshal(&legacyParams)
	require.NoError(t, err)

	result, err := adapter.UnmarshalFeeMarketParams(bz)
	require.NoError(t, err)
	require.True(t, result.NoBaseFee)
}

// A zero base_fee must convert to a zero Dec, not panic in LegacyNewDecFromInt.
func TestUnmarshalFeeMarketParamsZeroBaseFee(t *testing.T) {
	adapter, encCfg := newParamsAdapter()

	legacyParams := fmlegacytypes.Params{
		BaseFee:          sdkmath.ZeroInt(),
		MinGasPrice:      sdkmath.LegacyZeroDec(),
		MinGasMultiplier: sdkmath.LegacyZeroDec(),
	}

	bz, err := encCfg.Codec.Marshal(&legacyParams)
	require.NoError(t, err)

	result, err := adapter.UnmarshalFeeMarketParams(bz)
	require.NoError(t, err)
	require.True(t, result.BaseFee.IsZero())
}

func TestUnmarshalFeeMarketParamsInvalidBytes(t *testing.T) {
	adapter, _ := newParamsAdapter()

	_, err := adapter.UnmarshalFeeMarketParams([]byte{0xFF, 0xFE, 0xFD})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unmarshal legacy feemarket params")
}
