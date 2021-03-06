/*

  Copyright 2017 Loopring Project Ltd (Loopring Foundation).

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

*/

package ethaccessor_test

import (
	"fmt"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/test"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

var (
	version              = test.Version
	registerTokenAddress = "0x8b62ff4ddc9baeb73d0a3ea49d43e4fe8492935a"
	registerTokenSymbol  = "wrdn"
	account1             = test.Entity().Accounts[0].Address
	account2             = test.Entity().Accounts[1].Address
	lrcTokenAddress      = util.AllTokens["LRC"].Protocol
	wethTokenAddress     = util.AllTokens["WETH"].Protocol
	delegateAddress      = test.Delegate()
)

func TestEthNodeAccessor_SetTokenBalance(t *testing.T) {
	owner := account2
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000000))
	test.SetTokenBalance("LRC", owner, amount)
}

func TestEthNodeAccessor_Erc20Balance(t *testing.T) {
	accessor, err := test.GenerateAccessor()
	if err != nil {
		t.Fatalf("generate accessor error:%s", err.Error())
	}

	owner := account2
	tokenAddress := lrcTokenAddress
	balance, err := accessor.Erc20Balance(tokenAddress, owner, "latest")
	if err != nil {
		t.Fatalf("accessor get erc20 balance error:%s", err.Error())
	}

	t.Log(new(big.Rat).SetFrac(balance, big.NewInt(1e18)).FloatString(2))
}

func TestEthNodeAccessor_Approval(t *testing.T) {
	account := accounts.Account{Address: account2}

	tokenAddress := wethTokenAddress
	amount, _ := new(big.Int).SetString("100000000000000000000", 0) // 100weth
	spender := delegateAddress

	accessor, _ := test.GenerateAccessor()
	callMethod := accessor.ContractSendTransactionMethod(accessor.Erc20Abi, tokenAddress)
	if result, err := callMethod(account, "approve", nil, nil, nil, spender, amount); nil != err {
		t.Fatalf("call method approve error:%s", err.Error())
	} else {
		t.Logf("approve result:%s", result)
	}
}

func TestEthNodeAccessor_Allowance(t *testing.T) {
	accessor, err := test.GenerateAccessor()
	if err != nil {
		t.Fatalf("generate accessor error:%s", err.Error())
	}

	owner := account2
	tokenAddress := wethTokenAddress
	spender := delegateAddress

	if allowance, err := accessor.Erc20Allowance(tokenAddress, owner, spender, "latest"); err != nil {
		t.Fatalf("accessor get erc20 approval error:%s", err.Error())
	} else {
		t.Log(allowance.String())
	}
}

func TestEthNodeAccessor_CancelOrder(t *testing.T) {
	var (
		model           *dao.Order
		state           types.OrderState
		err             error
		result          string
		orderhash       = common.HexToHash("0x38a4ff508746feddb75e1652e2c430e23a179f949f331ebe81e624801b4b6f4d")
		cancelAmount, _ = new(big.Int).SetString("1000000000000000000000", 0)
	)

	// load config
	c := test.Cfg()

	// get order
	rds := test.Rds()
	if model, err = rds.GetOrderByHash(orderhash); err != nil {
		t.Fatalf(err.Error())
	}
	if err := model.ConvertUp(&state); err != nil {
		t.Fatalf(err.Error())
	}

	account := accounts.Account{Address: state.RawOrder.Owner}

	// create cancel order contract function parameters
	addresses := [3]common.Address{state.RawOrder.Owner, state.RawOrder.TokenS, state.RawOrder.TokenB}
	values := [7]*big.Int{state.RawOrder.AmountS, state.RawOrder.AmountB, state.RawOrder.Timestamp, state.RawOrder.Ttl, state.RawOrder.Salt, state.RawOrder.LrcFee, cancelAmount}
	buyNoMoreThanB := state.RawOrder.BuyNoMoreThanAmountB
	marginSplitPercentage := state.RawOrder.MarginSplitPercentage
	v := state.RawOrder.V
	s := state.RawOrder.S
	r := state.RawOrder.R

	// call cancel order
	accessor, _ := test.GenerateAccessor()
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	callMethod := accessor.ContractSendTransactionMethod(accessor.ProtocolImplAbi, protocol)
	if result, err = callMethod(account, "cancelOrder", big.NewInt(200000), big.NewInt(21000000000), nil, addresses, values, buyNoMoreThanB, marginSplitPercentage, v, r, s); nil != err {
		t.Fatalf("call method cancelOrder error:%s", err.Error())
	} else {
		t.Logf("cancelOrder result:%s", result)
	}
}

func TestEthNodeAccessor_GetCancelledOrFilled(t *testing.T) {
	c := test.Cfg()
	accessor, _ := test.GenerateAccessor()
	orderhash := common.HexToHash("0x77aecf96a71d260074ab6ad9352365f0e83cd87d4d3a424071e76cafc393f549")

	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	if amount, err := accessor.GetCancelledOrFilled(protocol, orderhash, "latest"); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("cancelOrFilled amount:%s", amount.String())
	}
}

// cutoff的值必须在两个块的timestamp之间
func TestEthNodeAccessor_Cutoff(t *testing.T) {
	c := test.Cfg()
	account := accounts.Account{Address: account2}
	cutoff := big.NewInt(1518700280)

	accessor, _ := test.GenerateAccessor()
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	callMethod := accessor.ContractSendTransactionMethod(accessor.ProtocolImplAbi, protocol)
	if result, err := callMethod(account, "setCutoff", nil, nil, nil, cutoff); nil != err {
		t.Fatalf("call method setCutoff error:%s", err.Error())
	} else {
		t.Logf("cutoff result:%s", result)
	}
}

func TestEthNodeAccessor_GetCutoff(t *testing.T) {
	c := test.Cfg()
	accessor, _ := test.GenerateAccessor()

	owner := account1
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	if timestamp, err := accessor.GetCutoff(protocol, owner, "latest"); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("cutoff timestamp:%s", timestamp.String())
	}
}

func TestEthNodeAccessor_TokenRegister(t *testing.T) {
	c := test.Cfg()
	account := accounts.Account{Address: common.HexToAddress(c.Miner.Miner)}

	accessor, _ := test.GenerateAccessor()
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	callMethod := accessor.ContractSendTransactionMethod(accessor.TokenRegistryAbi, accessor.ProtocolAddresses[protocol].TokenRegistryAddress)
	if result, err := callMethod(account, "registerToken", nil, nil, nil, common.HexToAddress(registerTokenAddress), registerTokenSymbol); nil != err {
		t.Fatalf("call method registerToken error:%s", err.Error())
	} else {
		t.Logf("registerToken result:%s", result)
	}
}

func TestEthNodeAccessor_TokenUnRegister(t *testing.T) {
	c := test.Cfg()
	account := accounts.Account{Address: common.HexToAddress(c.Miner.Miner)}

	accessor, _ := test.GenerateAccessor()
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	callMethod := accessor.ContractSendTransactionMethod(accessor.TokenRegistryAbi, accessor.ProtocolAddresses[protocol].TokenRegistryAddress)
	if result, err := callMethod(account, "unregisterToken", nil, nil, nil, common.HexToAddress(registerTokenAddress), registerTokenSymbol); nil != err {
		t.Fatalf("call method unregisterToken error:%s", err.Error())
	} else {
		t.Logf("unregisterToken result:%s", result)
	}
}

func TestEthNodeAccessor_GetAddressBySymbol(t *testing.T) {
	c := test.Cfg()
	accessor, _ := test.GenerateAccessor()

	var result string
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	callMethod := accessor.ContractCallMethod(accessor.TokenRegistryAbi, accessor.ProtocolAddresses[protocol].TokenRegistryAddress)
	if err := callMethod(&result, "getAddressBySymbol", "latest", registerTokenSymbol); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("symbol map:%s->%s", registerTokenSymbol, common.HexToAddress(result).Hex())
	}
}

// 注册合约
func TestEthNodeAccessor_AuthorizedAddress(t *testing.T) {
	c := test.Cfg()
	account := accounts.Account{Address: common.HexToAddress(c.Miner.Miner)}

	accessor, _ := test.GenerateAccessor()
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	callMethod := accessor.ContractSendTransactionMethod(accessor.DelegateAbi, accessor.ProtocolAddresses[protocol].DelegateAddress)
	if result, err := callMethod(account, "authorizeAddress", nil, nil, nil, protocol); nil != err {
		t.Fatalf("call method authorizeAddress error:%s", err.Error())
	} else {
		t.Logf("authorizeAddress result:%s", result)
	}
}

func TestEthNodeAccessor_DeAuthorizedAddress(t *testing.T) {
	c := test.Cfg()
	account := accounts.Account{Address: common.HexToAddress(c.Miner.Miner)}

	accessor, _ := test.GenerateAccessor()
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	callMethod := accessor.ContractSendTransactionMethod(accessor.DelegateAbi, accessor.ProtocolAddresses[protocol].DelegateAddress)
	if result, err := callMethod(account, "deauthorizeAddress", nil, nil, nil, protocol); nil != err {
		t.Fatalf("call method deauthorizeAddress error:%s", err.Error())
	} else {
		t.Logf("deauthorizeAddress result:%s", result)
	}
}

func TestEthNodeAccessor_IsAddressAuthorized(t *testing.T) {
	c := test.Cfg()
	accessor, _ := test.GenerateAccessor()

	var result string
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	callMethod := accessor.ContractCallMethod(accessor.DelegateAbi, accessor.ProtocolAddresses[protocol].DelegateAddress)
	if err := callMethod(&result, "isAddressAuthorized", "latest", protocol); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("symbol map:%s->%s", registerTokenSymbol, result)
	}
}

func TestEthNodeAccessor_WethDeposit(t *testing.T) {
	account := accounts.Account{Address: account1}

	wethAddr := wethTokenAddress
	amount := new(big.Int).SetInt64(1000000)
	accessor, _ := test.GenerateAccessor()
	callMethod := accessor.ContractSendTransactionMethod(accessor.WethAbi, wethAddr)
	if result, err := callMethod(account, "deposit", big.NewInt(200000), big.NewInt(21000000000), amount); nil != err {
		t.Fatalf("call method weth-deposit error:%s", err.Error())
	} else {
		t.Logf("weth-deposit result:%s", result)
	}
}

func TestEthNodeAccessor_WethWithdrawal(t *testing.T) {
	account := accounts.Account{Address: account1}

	wethAddr := wethTokenAddress
	amount, _ := new(big.Int).SetString("100", 0)
	accessor, _ := test.GenerateAccessor()
	callMethod := accessor.ContractSendTransactionMethod(accessor.WethAbi, wethAddr)
	if result, err := callMethod(account, "withdraw", big.NewInt(200000), big.NewInt(21000000000), nil, amount); nil != err {
		t.Fatalf("call method weth-withdraw error:%s", err.Error())
	} else {
		t.Logf("weth-withdraw result:%s", result)
	}
}

func TestEthNodeAccessor_WethTransfer(t *testing.T) {
	account := accounts.Account{Address: account1}

	wethAddr := wethTokenAddress
	amount := new(big.Int).SetInt64(100)
	to := account2
	accessor, _ := test.GenerateAccessor()
	callMethod := accessor.ContractSendTransactionMethod(accessor.WethAbi, wethAddr)
	if result, err := callMethod(account, "transfer", big.NewInt(200000), big.NewInt(21000000000), nil, to, amount); nil != err {
		t.Fatalf("call method weth-transfer error:%s", err.Error())
	} else {
		t.Logf("weth-transfer result:%s", result)
	}
}

func TestEthNodeAccessor_TokenAddress(t *testing.T) {
	c := test.Cfg()

	symbol := "WETH"
	accessor, _ := test.GenerateAccessor()
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	callMethod := accessor.ContractCallMethod(accessor.TokenRegistryAbi, protocol)
	var result string
	if err := callMethod(&result, "getAddressBySymbol", "latest", symbol); nil != err {
		t.Fatalf("call method tokenAddress error:%s", err.Error())
	} else {
		t.Logf("symbol:%s-> address:%s", symbol, result)
	}
}

// 10个块性能测试:
// batch   15~33秒之间
// single  240秒以上
func TestEthNodeAccessor_BatchTransactions(t *testing.T) {
	accessor, _ := test.GenerateAccessor()
	start := 4884729
	end := start + 2
	iterator := accessor.BlockIterator(big.NewInt(int64(start)), big.NewInt(int64(end)), false, uint64(0))

	testcase := "batch"

	for blocknumber := start; blocknumber < end; blocknumber++ {
		inter, err := iterator.Next()
		if err != nil {
			t.Fatalf(err.Error())
		}

		block := inter.(*ethaccessor.BlockWithTxHash)
		i := 0

		switch testcase {
		case "batch":
			var reqs []*ethaccessor.BatchTransactionReq
			for _, txstr := range block.Transactions {
				var (
					req ethaccessor.BatchTransactionReq
					tx  ethaccessor.Transaction
					err error
				)
				req.TxHash = txstr
				req.TxContent = tx
				req.Err = err
				reqs = append(reqs, &req)
			}

			if err := accessor.BatchTransactions(1, reqs); err != nil {
				t.Fatalf(err.Error())
			}

			for _, v := range reqs {
				if v.Err != nil {
					t.Fatalf(err.Error())
				} else {
					fmt.Printf("tx hash:%s, blocknumber:%d \r\n", v.TxContent.Hash, v.TxContent.BlockNumber.Int())
					i++
				}
			}
			fmt.Printf("block %d has %d transations and successed %d \r\n", blocknumber, len(reqs), i)
			break

		case "single":
			for _, txstr := range block.Transactions {
				var tx ethaccessor.Transaction
				if err := accessor.Call(&tx, "eth_getTransactionByHash", txstr); err != nil {
					fmt.Errorf(err.Error())
				} else {
					fmt.Printf("tx hash:%s, blocknumber:%d \r\n", tx.Hash, tx.BlockNumber.Int())
					i++
				}
			}

			fmt.Printf("block %d has %d transations and successed %d \r\n", blocknumber, len(block.Transactions), i)
			break
		}

	}

}

func TestEthNodeAccessor_BatchTransactionRecipients(t *testing.T) {
	accessor, _ := test.GenerateAccessor()
	start := 4884729
	end := start + 2
	iterator := accessor.BlockIterator(big.NewInt(int64(start)), big.NewInt(int64(end)), false, uint64(0))

	for blocknumber := start; blocknumber < end; blocknumber++ {
		inter, err := iterator.Next()
		if err != nil {
			t.Fatalf(err.Error())
		}

		block := inter.(*ethaccessor.BlockWithTxHash)
		i := 0

		var reqs []*ethaccessor.BatchTransactionRecipientReq
		for _, txstr := range block.Transactions {
			var (
				req ethaccessor.BatchTransactionRecipientReq
				tx  ethaccessor.TransactionReceipt
				err error
			)
			req.TxHash = txstr
			req.TxContent = tx
			req.Err = err
			reqs = append(reqs, &req)
		}

		if err := accessor.BatchTransactionRecipients(1, reqs); err != nil {
			t.Fatalf(err.Error())
		}

		for _, v := range reqs {
			if v.Err != nil {
				t.Fatalf(err.Error())
			} else {
				fmt.Printf("tx hash:%s, blocknumber:%d \r\n", v.TxContent.TransactionHash, v.TxContent.BlockNumber.Int())
				i++
			}
		}
		fmt.Printf("block %d has %d transations and successed %d \r\n", blocknumber, len(reqs), i)

	}

}
