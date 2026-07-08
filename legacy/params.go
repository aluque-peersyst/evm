package legacy

import (
	"fmt"
	"strconv"
	"strings"

	evmlegacytypes "github.com/cosmos/evm/rpc/types/legacy"
	fmlegacytypes "github.com/cosmos/evm/rpc/types/legacy/feemarket"
	fmtypes "github.com/cosmos/evm/x/feemarket/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	sdkmath "cosmossdk.io/math"
)

const legacyExtraEIPPrefix = "ethereum_"

// UnmarshalLegacyParams converts pre-v9 (Ethermint) EVM params bytes to the
// current schema.
func (l *LegacyAdapter) UnmarshalLegacyParams(bz []byte) (evmtypes.Params, error) {
	var legacyParams evmlegacytypes.Params
	if err := l.cdc.Unmarshal(bz, &legacyParams); err != nil {
		return evmtypes.Params{}, fmt.Errorf("unmarshal legacy EVM params: %w", err)
	}

	// "ethereum_3855" → 3855, mirroring the v9 upgrade migration.
	eips := make([]int64, len(legacyParams.ExtraEIPs))
	for i, extraEIP := range legacyParams.ExtraEIPs {
		sanitized := strings.TrimPrefix(extraEIP, legacyExtraEIPPrefix)
		intEIP, err := strconv.ParseInt(sanitized, 10, 64)
		if err != nil {
			return evmtypes.Params{}, fmt.Errorf("invalid legacy ExtraEIP %q: %w", extraEIP, err)
		}
		eips[i] = intEIP
	}

	accessControl := evmtypes.AccessControl{
		Create: evmtypes.AccessControlType{
			AccessType:        evmtypes.AccessType(legacyParams.AccessControl.Create.AccessType),
			AccessControlList: legacyParams.AccessControl.Create.AccessControlList,
		},
		Call: evmtypes.AccessControlType{
			AccessType:        evmtypes.AccessType(legacyParams.AccessControl.Call.AccessType),
			AccessControlList: legacyParams.AccessControl.Call.AccessControlList,
		},
	}

	return evmtypes.Params{
		EvmDenom:                legacyParams.EvmDenom,
		ExtraEIPs:               eips,
		EVMChannels:             legacyParams.EVMChannels,
		AccessControl:           accessControl,
		ActiveStaticPrecompiles: legacyParams.ActiveStaticPrecompiles,
	}, nil
}

// UnmarshalFeeMarketParams converts pre-v9 (Ethermint) feemarket params bytes
// to the current schema. The encodings are wire-compatible (current-schema
// decoding silently rescales base_fee by 10^-18) so the height dispatch is
// the only safeguard.
func (l *LegacyAdapter) UnmarshalFeeMarketParams(bz []byte) (fmtypes.Params, error) {
	var legacyParams fmlegacytypes.Params
	if err := l.cdc.Unmarshal(bz, &legacyParams); err != nil {
		return fmtypes.Params{}, fmt.Errorf("unmarshal legacy feemarket params: %w", err)
	}

	return fmtypes.Params{
		NoBaseFee:                legacyParams.NoBaseFee,
		BaseFeeChangeDenominator: legacyParams.BaseFeeChangeDenominator,
		ElasticityMultiplier:     legacyParams.ElasticityMultiplier,
		EnableHeight:             legacyParams.EnableHeight,
		BaseFee:                  sdkmath.LegacyNewDecFromInt(legacyParams.BaseFee),
		MinGasPrice:              legacyParams.MinGasPrice,
		MinGasMultiplier:         legacyParams.MinGasMultiplier,
	}, nil
}
