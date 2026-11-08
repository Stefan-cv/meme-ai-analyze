package risk

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Oracle manages on-chain report anchoring via the RiskReportOracle contract.
type Oracle struct {
	contractAddress common.Address
	privateKey      *ecdsa.PrivateKey
	client          *ethclient.Client
	chainID         *big.Int
}

// NewOracle creates a new Oracle instance.
func NewOracle(contractAddrHex, privateKeyHex, rpcURL string) *Oracle {
	o := &Oracle{
		contractAddress: common.HexToAddress(contractAddrHex),
		chainID:         big.NewInt(56), // BSC default
	}

	if privateKeyHex != "" {
		key, err := crypto.HexToECDSA(privateKeyHex)
		if err != nil {
			log.Printf("[Oracle] Warning: invalid private key: %v", err)
		} else {
			o.privateKey = key
		}
	}

	if rpcURL != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			client, err := ethclient.DialContext(ctx, rpcURL)
			if err != nil {
				log.Printf("[Oracle] Warning: failed to connect to RPC: %v", err)
			} else {
				o.client = client
				cID, cErr := client.ChainID(ctx)
				if cErr == nil {
					o.chainID = cID
				}
			}
		}

	return o
}

// SubmitReport anchors a risk report hash on-chain.
// Returns the transaction hash or an error.
func (o *Oracle) SubmitReport(ctx context.Context, tokenAddr common.Address, reportHash [32]byte, riskScore uint8, version string) (string, error) {
	if o.client == nil {
		return "", fmt.Errorf("oracle: no RPC client available")
	}
	if o.privateKey == nil {
		return "", fmt.Errorf("oracle: no private key configured")
	}
	if o.contractAddress == (common.Address{}) {
		return "", fmt.Errorf("oracle: no contract address configured")
	}

	// In production:
	//   1. Create a signed transaction calling riskReportOracle.submitReport()
	//   2. Use abigen-generated Go bindings for the contract
	//   3. Estimate gas, set gas price, sign, and broadcast
	//
	// For now, return a simulated tx hash
	log.Printf("[Oracle] Would submit report for %s (score=%d, version=%s)",
		tokenAddr.Hex(), riskScore, version)

	txHash := crypto.Keccak256Hash([]byte(fmt.Sprintf("report:%s:%d:%s", tokenAddr.Hex(), riskScore, version)))
	return txHash.Hex(), nil
}

// RecordScan records a scan event on-chain (permissionless).
func (o *Oracle) RecordScan(ctx context.Context, tokenAddr common.Address) (string, error) {
	if o.client == nil {
		return "", fmt.Errorf("oracle: no RPC client available")
	}

	// In production: call riskReportOracle.recordScan(tokenAddr)
	log.Printf("[Oracle] Would record scan for %s", tokenAddr.Hex())

	txHash := crypto.Keccak256Hash([]byte(fmt.Sprintf("scan:%s", tokenAddr.Hex())))
	return txHash.Hex(), nil
}

// GetAddress returns the oracle's wallet address (derived from private key).
func (o *Oracle) GetAddress() common.Address {
	if o.privateKey == nil {
		return common.Address{}
	}
	return crypto.PubkeyToAddress(o.privateKey.PublicKey)
}

// IsConfigured returns true if the oracle is ready to submit reports.
func (o *Oracle) IsConfigured() bool {
	return o.client != nil &&
		o.privateKey != nil &&
		o.contractAddress != (common.Address{})
}

// ReportHash computes the keccak256 hash of a report for on-chain anchoring.
func ReportHash(jsonData []byte) [32]byte {
	return crypto.Keccak256Hash(jsonData)
}

// HexToBytes32 converts a hex string to a [32]byte.
func HexToBytes32(hexStr string) ([32]byte, error) {
	var result [32]byte
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return result, err
	}
	copy(result[:], data)
	return result, nil
}
