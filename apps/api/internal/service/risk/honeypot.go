package risk

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// HoneypotDetector checks whether a token is a honeypot via eth_call simulation.
type HoneypotDetector struct {
	client *ethclient.Client
}

// NewHoneypotDetector creates a new HoneypotDetector instance.
func NewHoneypotDetector(rpcURL string) *HoneypotDetector {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Printf("[HoneypotDetector] Warning: failed to connect to RPC: %v", err)
	}
	return &HoneypotDetector{client: client}
}

// SetClient allows sharing a client.
func (d *HoneypotDetector) SetClient(client *ethclient.Client) {
	d.client = client
}

// Check performs a full honeypot check on a token.
// Returns: isHoneypot (bool), reason (string).
func (d *HoneypotDetector) Check(ctx context.Context, tokenAddr common.Address) (bool, string) {
	if d.client == nil {
		return false, "no RPC client"
	}

	// Strategy 1: Check if sell is blocked via PancakeSwap simulation
	if blocked, reason := d.checkSellBlocked(ctx, tokenAddr); blocked {
		return true, reason
	}

	// Strategy 2: Check for abnormally high transfer tax
	if highTax, reason := d.CheckTransferTax(ctx, tokenAddr); highTax {
		return true, reason
	}

	return false, ""
}

// checkSellBlocked simulates a token→WBNB swap via PancakeSwap Router.
func (d *HoneypotDetector) checkSellBlocked(ctx context.Context, tokenAddr common.Address) (bool, string) {
	// Get pair address
	callData, _ := factoryABI.Pack("getPair", tokenAddr, wbnbAddress)
	msg := ethereum.CallMsg{To: &pancakeSwapFactory, Data: callData}

	rpcCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	result, err := d.client.CallContract(rpcCtx, msg, nil)
	if err != nil || len(result) < 32 {
		return false, ""
	}
	pairAddr := common.BytesToAddress(result)
	if pairAddr == (common.Address{}) {
		return false, ""
	}

	// Simulate a small sell: swapExactTokensForETHSupportingFeeOnTransferTokens
	amountIn := big.NewInt(1000)
	deadline := big.NewInt(time.Now().Unix() + 300)

	// Build swap calldata manually for swapExactTokensForETHSupportingFeeOnTransferTokens
	// selector: 0x791ac947
	// params: amountIn, amountOutMin(0), path, to, deadline
	selector := common.Hex2Bytes("791ac947")

	// Pack: uint256, uint256, address[], address, uint256
	packedData := selector
	packedData = append(packedData, common.LeftPadBytes(amountIn.Bytes(), 32)...)
	packedData = append(packedData, common.LeftPadBytes(big.NewInt(0).Bytes(), 32)...)

	// path offset (pointer to array data)
	pathOffset := big.NewInt(0x80) // after 5 x 32-byte fields
	packedData = append(packedData, common.LeftPadBytes(pathOffset.Bytes(), 32)...)

	// to address (pair address as recipient)
	packedData = append(packedData, common.LeftPadBytes(pairAddr.Bytes(), 32)...)

	// deadline
	packedData = append(packedData, common.LeftPadBytes(deadline.Bytes(), 32)...)

	// array length (2 addresses)
	packedData = append(packedData, common.LeftPadBytes(big.NewInt(2).Bytes(), 32)...)
	// token address
	packedData = append(packedData, common.LeftPadBytes(tokenAddr.Bytes(), 32)...)
	// wbnb address
	packedData = append(packedData, common.LeftPadBytes(wbnbAddress.Bytes(), 32)...)

	// Simulate from pair address (as if pair is calling the router)
	routerMsg := ethereum.CallMsg{
		To:   &pancakeSwapRouter,
		Data: packedData,
		From: pairAddr,
	}

	result2, err := d.client.CallContract(rpcCtx, routerMsg, nil)
	if err != nil || len(result2) == 0 {
		log.Printf("[HoneypotDetector] Sell simulation reverted → possible honeypot: %v", err)
		return true, "⚠️ 模拟卖出失败，可能为蜜罐代币"
	}

	return false, ""
}

// CheckTransferTax simulates a transfer to detect tax rate.
func (d *HoneypotDetector) CheckTransferTax(ctx context.Context, tokenAddr common.Address) (bool, string) {
	rpcCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Simulate transfer of 1000 tokens from pair to dead address
	amount := big.NewInt(1000)
	deadAddr := common.HexToAddress("0x000000000000000000000000000000000000dEaD")

	// Get pair address to use as sender
	callData, _ := factoryABI.Pack("getPair", tokenAddr, wbnbAddress)
	msg := ethereum.CallMsg{To: &pancakeSwapFactory, Data: callData}
	result, err := d.client.CallContract(rpcCtx, msg, nil)
	if err != nil || len(result) < 32 {
		return false, ""
	}
	pairAddr := common.BytesToAddress(result)
	if pairAddr == (common.Address{}) {
		return false, ""
	}

	// Check pair's balance before (simulated)
	balData, _ := erc20ABI.Pack("balanceOf", pairAddr)
	balMsg := ethereum.CallMsg{To: &tokenAddr, Data: balData}
	balResult, err := d.client.CallContract(rpcCtx, balMsg, nil)
	if err != nil {
		return false, ""
	}

	// Simulate transfer from pair to dead
	transferData, _ := erc20ABI.Pack("transfer", deadAddr, amount)
	transferMsg := ethereum.CallMsg{
		To:   &tokenAddr,
		Data: transferData,
		From: pairAddr,
	}
	_, err = d.client.CallContract(rpcCtx, transferMsg, nil)
	if err != nil {
		log.Printf("[HoneypotDetector] Transfer reverted → possible honeypot: %v", err)
		return true, "⚠️ 转账失败，合约可能阻止卖出"
	}

	// If balance is too low to transfer, not necessarily a tax issue
	if len(balResult) >= 32 {
		bal := new(big.Int).SetBytes(balResult)
		if bal.Cmp(amount) < 0 {
			return false, "" // not enough balance
		}
	}

	// Check dead address balance after (should be exactly amount minus tax)
	// This gives us the actual received amount
	deadBalData, _ := erc20ABI.Pack("balanceOf", deadAddr)
	deadBalMsg := ethereum.CallMsg{To: &tokenAddr, Data: deadBalData}
	deadBalResult, err := d.client.CallContract(rpcCtx, deadBalMsg, nil)
	if err != nil {
		return false, ""
	}

	if len(deadBalResult) >= 32 {
		received := new(big.Int).SetBytes(deadBalResult)
		if received.Cmp(big.NewInt(0)) == 0 {
			return true, "⚠️ 代币有 100% 交易税，无法收到代币"
		}
		if received.Cmp(amount) < 0 {
			// Calculate tax rate
			diff := new(big.Int).Sub(amount, received)
			taxRate := float64(diff.Uint64()) / float64(amount.Uint64()) * 100
			if taxRate > 50 {
				return true, fmt.Sprintf("⚠️ 交易税过高 (%.0f%%)", taxRate)
			}
			log.Printf("[HoneypotDetector] Transfer tax: %.0f%%", taxRate)
		}
	}

	return false, ""
}
