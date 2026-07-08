// Package legacy reads EVM state written in layouts that predate the v9
// Ethermint → cosmos/evm migration.
package legacy

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/cosmos/evm/x/vm/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// V9 is the upgrade that migrated Ethermint state layouts to cosmos/evm.
const V9 = "v9.0.0"

// UpgradeKeeper is the subset of the x/upgrade keeper used by LegacyAdapter.
type UpgradeKeeper interface {
	GetDoneHeight(ctx context.Context, name string) (int64, error)
}

// AccountKeeper is the subset of the auth keeper used by LegacyAdapter.
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}

// codeHasher is implemented by the consuming app's legacy EthAccount type.
type codeHasher interface {
	GetCodeHash() common.Hash
}

// LegacyAdapter reads EVM state written in the pre-v9 (Ethermint) layout.
type LegacyAdapter struct {
	cdc      codec.BinaryCodec
	uk       UpgradeKeeper
	ak       AccountKeeper
	v9Height int64
	loaded   bool
}

// NewLegacyAdapter returns an unloaded adapter; call Load once the store is loaded.
func NewLegacyAdapter(cdc codec.BinaryCodec, uk UpgradeKeeper, ak AccountKeeper) *LegacyAdapter {
	return &LegacyAdapter{cdc: cdc, uk: uk, ak: ak}
}

// Load reads the v9 upgrade height from the state mounted in ctx. Zero means
// the chain has no pre-v9 history.
func (l *LegacyAdapter) Load(ctx sdk.Context) error {
	height, err := l.uk.GetDoneHeight(ctx, V9)
	if err != nil {
		return fmt.Errorf("legacy: read v9 upgrade height: %w", err)
	}
	l.v9Height = height
	l.loaded = true
	return nil
}

// IsLegacy reports whether the state mounted in ctx predates the v9 migration.
// It panics if called before Load.
func (l *LegacyAdapter) IsLegacy(ctx sdk.Context) bool {
	if !l.loaded {
		panic("legacy: IsLegacy called before Load")
	}
	return ctx.BlockHeight() < l.v9Height
}

// GetCodeHash reads a contract's code hash from the legacy EthAccount.
func (l *LegacyAdapter) GetCodeHash(ctx sdk.Context, addr common.Address) common.Hash {
	acct := l.ak.GetAccount(ctx, sdk.AccAddress(addr.Bytes()))
	if legacyAcct, ok := acct.(codeHasher); ok {
		if h := legacyAcct.GetCodeHash(); !types.IsEmptyCodeHash(h.Bytes()) {
			return h
		}
	}
	return common.BytesToHash(types.EmptyCodeHash)
}
