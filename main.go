package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type Faucet struct {
	client        *ethclient.Client
	privateKey    *ecdsa.PrivateKey
	fromAddress   common.Address
	chainID       *big.Int
	amount        *big.Int
	usedAddresses map[string]time.Time
	mu            sync.RWMutex
	cooldown      time.Duration
}

type Request struct {
	Address string `json:"address" binding:"required"`
}

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	TxHash  string `json:"tx_hash,omitempty"`
}

func NewFaucet() (*Faucet, error) {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// Connect to Ethereum network
	rpcURL := os.Getenv("RPC_URL")
	if rpcURL == "" {
		rpcURL = "http://localhost:8545" // Default to local node
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum network: %v", err)
	}

	// Get private key from environment
	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		return nil, fmt.Errorf("PRIVATE_KEY environment variable is required")
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	// Get chain ID
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %v", err)
	}

	// Get amount from environment (in wei)
	amountStr := os.Getenv("FAUCET_AMOUNT")
	if amountStr == "" {
		amountStr = "1000000000000000000" // 1 ETH default
	}
	amount, ok := new(big.Int).SetString(amountStr, 10)
	if !ok {
		return nil, fmt.Errorf("invalid FAUCET_AMOUNT")
	}

	// Get cooldown from environment
	cooldownStr := os.Getenv("COOLDOWN_HOURS")
	if cooldownStr == "" {
		cooldownStr = "24" // 24 hours default
	}
	cooldownHours, err := strconv.Atoi(cooldownStr)
	if err != nil {
		return nil, fmt.Errorf("invalid COOLDOWN_HOURS")
	}

	return &Faucet{
		client:        client,
		privateKey:    privateKey,
		fromAddress:   fromAddress,
		chainID:       chainID,
		amount:        amount,
		usedAddresses: make(map[string]time.Time),
		cooldown:      time.Duration(cooldownHours) * time.Hour,
	}, nil
}

func (f *Faucet) SendTokens(toAddress string) (*types.Transaction, error) {
	// Validate address
	if !common.IsHexAddress(toAddress) {
		return nil, fmt.Errorf("invalid Ethereum address")
	}

	// Check if address has been used recently
	f.mu.RLock()
	lastUsed, exists := f.usedAddresses[toAddress]
	f.mu.RUnlock()

	if exists && time.Since(lastUsed) < f.cooldown {
		remaining := f.cooldown - time.Since(lastUsed)
		return nil, fmt.Errorf("address %s can request again in %v", toAddress, remaining.Round(time.Minute))
	}

	// Get nonce
	nonce, err := f.client.PendingNonceAt(context.Background(), f.fromAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %v", err)
	}

	// Get gas price
	gasPrice, err := f.client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %v", err)
	}

	// Prepare call message to estimate gas for the actual send transaction
	to := common.HexToAddress(toAddress)
	gas, err := f.client.EstimateGas(context.Background(), ethereum.CallMsg{
		From:  f.fromAddress,
		To:    &to,
		Value: f.amount,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to estimate gas: %v", err)
	}

	// Create transaction
	tx := types.NewTransaction(
		nonce,
		common.HexToAddress(toAddress),
		f.amount,
		gas,
		gasPrice,
		nil,
	)

	// Sign transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(f.chainID), f.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Send transaction
	err = f.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %v", err)
	}

	// Mark address as used
	f.mu.Lock()
	f.usedAddresses[toAddress] = time.Now()
	f.mu.Unlock()

	return signedTx, nil
}

func (f *Faucet) GetBalance() (*big.Int, error) {
	balance, err := f.client.BalanceAt(context.Background(), f.fromAddress, nil)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

func main() {
	faucet, err := NewFaucet()
	if err != nil {
		log.Fatalf("Failed to create faucet: %v", err)
	}

	// Get balance
	balance, err := faucet.GetBalance()
	if err != nil {
		log.Printf("Warning: failed to get balance: %v", err)
	} else {
		log.Printf("Faucet address: %s", faucet.fromAddress.Hex())
		log.Printf("Faucet balance: %s ETH", new(big.Float).Quo(new(big.Float).SetInt(balance), new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))))
	}

	// Setup Gin router
	r := gin.Default()

	r.POST("/request", func(c *gin.Context) {
		var req Request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid request format",
			})
			return
		}

		// Send tokens
		tx, err := faucet.SendTokens(req.Address)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Success: true,
			Message: "Tokens sent successfully!",
			TxHash:  tx.Hash().Hex(),
		})
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting faucet server on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
