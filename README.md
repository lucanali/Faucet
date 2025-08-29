# Faucet

A simple and secure faucet built with Go that allows users to request testnet tokens. This faucet is designed for development and testing purposes on Custom networks.

## Features

- **Secure Token Distribution**: Sends tokens to requested addresses
- **Rate Limiting**: Configurable cooldown period to prevent abuse
- **RESTful API**: Simple HTTP endpoints for token requests
- **Environment Configuration**: Flexible configuration through environment variables
- **Transaction History**: Returns transaction hashes for all successful requests

## Prerequisites

- Go 1.21 or higher
- Access to an Ethereum RPC endpoint (local node or remote service)
- An Ethereum account with sufficient balance for faucet operations
- Private key for the faucet account

## Installation

1. Clone the repository:
```bash
git clone <your-repo-url>
cd faucet
```

2. Install dependencies:
```bash
go mod tidy
```

3. Copy the environment configuration:
```bash
cp env.example .env
```

4. Edit the `.env` file with your configuration (see Configuration section below)

## Configuration

Create a `.env` file in the project root with the following variables:

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `RPC_URL` | Ethereum RPC endpoint URL | `http://localhost:8545` | No |
| `PRIVATE_KEY` | Faucet account private key (without 0x prefix) | - | **Yes** |
| `FAUCET_AMOUNT` | Amount to send per request in wei | `1000000000000000000` (1 ETH) | No |
| `COOLDOWN_HOURS` | Hours before same address can request again | `24` | No |
| `PORT` | Server port | `8080` | No |

### Example Configuration

```env
RPC_URL=http://localhost:9650/ext/bc/2sBQUKZFdtgBMDhfNX9Ph72SPE8fXj33yY1a33vQ97wiQL76es/rpc
PRIVATE_KEY=your_private_key_here_without_0x_prefix
FAUCET_AMOUNT=1000000000000000000
COOLDOWN_HOURS=24
PORT=8080
```

**⚠️ Security Note**: Never commit your `.env` file or expose your private key. The `.env` file is already in `.gitignore`.

## Usage

### Starting the Faucet

1. Ensure your `.env` file is configured correctly
2. Run the faucet:
```bash
go run main.go
```

The faucet will start and display:
- Faucet address
- Server port

### API Endpoints

#### Request Tokens

**POST** `/request`

Request body:
```json
{
  "address": "0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6"
}
```

Response:
```json
{
  "success": true,
  "message": "Tokens sent successfully!",
  "tx_hash": "0x1234..."
}
```

Error response:
```json
{
  "success": false,
  "message": "Error description"
}
```

### Example Usage

#### Using curl
```bash
curl -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{"address": "0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6"}'
```

## Architecture

The faucet is built with a clean, modular architecture:

- **Faucet struct**: Manages the core faucet functionality
- **HTTP server**: Gin-based REST API for handling requests
- **Ethereum client**: Go Ethereum client for blockchain interactions
- **Rate limiting**: In-memory cooldown system (currently commented out)
- **Configuration**: Environment-based configuration management

## Security Features

- **Private Key Validation**: Ensures valid Ethereum private key format
- **Address Validation**: Validates Ethereum address format before processing
- **Rate Limiting**: Configurable cooldown periods (currently disabled)
- **Transaction Signing**: Secure transaction signing with EIP-155 support
- **Error Handling**: Comprehensive error handling and logging

## Development

### Project Structure
```
faucet/
├── main.go          # Main application entry point
├── go.mod           # Go module dependencies
├── go.sum           # Go module checksums
├── env.example      # Environment configuration template
└── README.md        # This file
```

### Building
```bash
go build -o faucet main.go
```

## Dependencies

- **github.com/ethereum/go-ethereum**: Ethereum client and crypto utilities
- **github.com/gin-gonic/gin**: HTTP web framework
- **github.com/joho/godotenv**: Environment variable management

## Troubleshooting

### Common Issues

1. **"PRIVATE_KEY environment variable is required"**
   - Ensure your `.env` file exists and contains the `PRIVATE_KEY` variable
   - Make sure the private key doesn't have the `0x` prefix

2. **"failed to connect to Ethereum network"**
   - Check your `RPC_URL` configuration
   - Ensure the RPC endpoint is accessible
   - Verify network connectivity

3. **"invalid private key"**
   - Ensure the private key is a valid 64-character hex string
   - Remove any `0x` prefix if present

4. **"failed to get chain ID"**
   - Check if your RPC endpoint supports the required methods
   - Verify the endpoint is for the correct network

### Logs

The faucet provides detailed logging for debugging:
- Faucet initialization details
- Transaction status
- Server startup information
