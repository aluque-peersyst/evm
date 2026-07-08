package keeper_test

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"

	"github.com/cosmos/evm/legacy"
	"github.com/cosmos/evm/legacy/legacytestutil"
	vmtypes "github.com/cosmos/evm/x/vm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// legacyAccount stands in for a pre-v9 Ethermint EthAccount.
type legacyAccount struct {
	*authtypes.BaseAccount
	codeHash common.Hash
}

func (a legacyAccount) GetCodeHash() common.Hash { return a.codeHash }

func (suite *KeeperTestSuite) TestGetCodeHash() {
	contractHash := common.HexToHash("0x1122334455667788990011223344556677889900112233445566778899001122")
	indexHash := common.HexToHash("0xaabbccddeeff00112233445566778899aabbccddeeff001122334455667788ff")
	emptyHash := common.BytesToHash(vmtypes.EmptyCodeHash)

	legacyCtx := suite.ctx.WithBlockHeight(legacytestutil.V9Height - 1)
	currentCtx := suite.ctx.WithBlockHeight(legacytestutil.V9Height)

	legacyWithCodeHash := func(addr common.Address, h common.Hash) sdk.AccountI {
		return legacyAccount{
			BaseAccount: authtypes.NewBaseAccountWithAddress(sdk.AccAddress(addr.Bytes())),
			codeHash:    h,
		}
	}

	testCases := []struct {
		name        string
		ctx         sdk.Context
		addr        common.Address
		wireAdapter bool
		upgraded    bool
		malleate    func(addr common.Address)
		expected    common.Hash
	}{
		{
			"should pass - current era: code hash present in index",
			currentCtx,
			common.HexToAddress("0x01"),
			true, true,
			func(addr common.Address) {
				suite.vmKeeper.SetCodeHash(suite.ctx, addr.Bytes(), contractHash.Bytes())
			},
			contractHash,
		},
		{
			"should fail - current era: index miss ignores legacy account",
			currentCtx,
			common.HexToAddress("0x02"),
			true, true,
			func(addr common.Address) {
				suite.accKeeper.On("GetAccount", mock.Anything, mock.Anything).
					Return(legacyWithCodeHash(addr, contractHash)).Maybe()
			},
			emptyHash,
		},
		{
			"should pass - legacy era: hash on legacy account, index ignored",
			legacyCtx,
			common.HexToAddress("0x03"),
			true, true,
			func(addr common.Address) {
				suite.vmKeeper.SetCodeHash(suite.ctx, addr.Bytes(), indexHash.Bytes())
				suite.accKeeper.On("GetAccount", mock.Anything, mock.Anything).
					Return(legacyWithCodeHash(addr, contractHash)).Maybe()
			},
			contractHash,
		},
		{
			"should fail - legacy era: legacy account with empty code hash",
			legacyCtx,
			common.HexToAddress("0x04"),
			true, true,
			func(addr common.Address) {
				suite.accKeeper.On("GetAccount", mock.Anything, mock.Anything).
					Return(legacyWithCodeHash(addr, emptyHash)).Maybe()
			},
			emptyHash,
		},
		{
			"should fail - legacy era: base account without code hash",
			legacyCtx,
			common.HexToAddress("0x05"),
			true, true,
			func(addr common.Address) {
				suite.accKeeper.On("GetAccount", mock.Anything, mock.Anything).
					Return(authtypes.NewBaseAccountWithAddress(sdk.AccAddress(addr.Bytes()))).Maybe()
			},
			emptyHash,
		},
		{
			"should fail - legacy era: account does not exist",
			legacyCtx,
			common.HexToAddress("0x06"),
			true, true,
			func(addr common.Address) {
				suite.accKeeper.On("GetAccount", mock.Anything, mock.Anything).
					Return(nil).Maybe()
			},
			emptyHash,
		},
		{
			"should pass - never-upgraded chain: index read at every height",
			legacyCtx,
			common.HexToAddress("0x07"),
			true, false,
			func(addr common.Address) {
				suite.vmKeeper.SetCodeHash(suite.ctx, addr.Bytes(), indexHash.Bytes())
				suite.accKeeper.On("GetAccount", mock.Anything, mock.Anything).
					Return(legacyWithCodeHash(addr, contractHash)).Maybe()
			},
			indexHash,
		},
		{
			"should pass - no adapter wired: index read at every height",
			legacyCtx,
			common.HexToAddress("0x09"),
			false, false,
			func(addr common.Address) {
				suite.vmKeeper.SetCodeHash(suite.ctx, addr.Bytes(), indexHash.Bytes())
			},
			indexHash,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.accKeeper.ExpectedCalls = nil

			var adapter *legacy.LegacyAdapter
			if tc.wireAdapter {
				var v9Height int64
				if tc.upgraded {
					v9Height = legacytestutil.V9Height
				}
				adapter = legacy.NewLegacyAdapter(nil, legacytestutil.FakeUpgradeKeeper{V9Height: v9Height}, suite.accKeeper)
				suite.Require().NoError(adapter.Load(currentCtx))
			}
			suite.vmKeeper.WithLegacyAdapter(adapter)

			tc.malleate(tc.addr)

			got := suite.vmKeeper.GetCodeHash(tc.ctx, tc.addr)
			suite.Require().Equal(tc.expected, got)
		})
	}
}
