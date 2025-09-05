# Sietch Vault

Sietch Vault is a decentralized file system that allows you to securely sync encrypted data across machines in low or no-internet conditions. Designed with the resilience of desert-dwellers in mind, it prioritizes portability, privacy, and robustness.

Sietch creates self-contained encrypted vaults that can sync over LAN, sneakernet USB drives, or weak WiFi networks. It operates fully offline, using chunked data, encryption, and peer-to-peer protocols to ensure your files are always protected and available - even when the internet is not.

## Motivation

Sietch Vault is designed for environments where:

- Internet is scarce or censored
- Data privacy is a necessity, not a feature
- People work nomadically - like researchers, journalists, or activists

It imagines what a file system would look like in a world more like Arrakis than San Francisco, with a focus on survival-first rather than cloud-first principles.

## Core Features

| Feature | Description |
|---------|-------------|
| AES256/GPG | Files are chunked and encrypted using strong symmetric/asymmetric keys |
| Offline Sync | Rsync-style syncing over TCP or LibP2P |
| Gossip Discovery | Lightweight peer discovery protocol for LAN environments |
| CLI First UX | Fast and minimal CLI to manage vaults and syncs |

## CI/CD Status

[![CI](https://github.com/substantialcattle5/sietch/actions/workflows/ci.yml/badge.svg)](https://github.com/substantialcattle5/sietch/actions/workflows/ci.yml)
[![Release](https://github.com/substantialcattle5/sietch/actions/workflows/release.yml/badge.svg)](https://github.com/substantialcattle5/sietch/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/substantialcattle5/sietch/branch/main/graph/badge.svg)](https://codecov.io/gh/substantialcattle5/sietch)
[![Go Report Card](https://goreportcard.com/badge/github.com/substantialcattle5/sietch)](https://goreportcard.com/report/github.com/substantialcattle5/sietch)

## How It Works

**Chunking**
Each file is split into chunks using configurable size (default 4MB). Identical chunks across files are deduplicated.

**Encryption**
Each chunk is encrypted before storage. You can:

- Use symmetric passphrase (AES-256-GCM)
- Use public/private keypairs [GPG-compatible](https://en.wikipedia.org/wiki/GNU_Privacy_Guard)

**Discovery**
Peers discover each other through:

- LAN gossip via UDP broadcast
- Manual IP whitelisting
- Future QR-code based sharing

**Syncing**
Inspired by rsync, Sietch only syncs:

- Missing chunks
- Changed metadata
- Securely over TCP, with optional compression

**Index Metadata**
Each sietch maintains an encrypted manifest (Merkle DAG) mapping chunk hashes to original files.

## Security Model

| Attack Vector | Mitigation |
|---------------|------------|
| Eavesdropping | Encrypted chunks over TLS or TCP |
| Vault tampering | Merkle trees and hash-based verification |
| Metadata leakage | Optional metadata obfuscation and encrypted indexes |
| Unauthorized sync | Public key signature verification for known devices |

## Installation

Full installation scripts and cross-platform builds will be provided in releases.

```
git clone https://github.com/SubstantialCattle5/Sietch.git
cd Sietch
go build
```

## Development Setup

### Prerequisites

- **Go 1.21+** - [Download](https://golang.org/dl/)
- **Node.js 16+** - [Download](https://nodejs.org/) (for Git hooks)
- **Git** - For version control

### Quick Start

1. **Clone the repository:**
   ```bash
   git clone https://github.com/substantialcattle5/sietch.git
   cd sietch
   ```

2. **Run the setup script:**
   ```bash
   ./scripts/setup-hooks.sh
   ```

   This will:
   - Install npm dependencies (Husky)
   - Install Go dependencies
   - Install development tools (golangci-lint, gosec)
   - Set up Git hooks for code quality
   - Run initial checks

3. **Verify setup:**
   ```bash
   make build    # Build the binary
   make test     # Run tests
   make lint     # Run linter
   ```

### Git Hooks

The setup includes pre-commit and pre-push hooks that automatically:

**Pre-commit checks:**
- ✅ Go code formatting (`go fmt`)
- ✅ Code linting (`golangci-lint`)
- ✅ Static analysis (`go vet`)
- ✅ Unit tests
- ✅ Conventional commit format

**Pre-push checks:**
- ✅ Full test suite (including integration tests)
- ✅ Build verification
- ✅ Security audit (`gosec`)

**Bypass hooks temporarily:**
```bash
HUSKY=0 git commit -m "bypass hooks"
git commit --no-verify -m "skip pre-commit"
git push --no-verify  # skip pre-push
```

### Available Commands

```bash
make help          # Show all available commands
make dev           # Development workflow (fmt, test, build)
make check         # Full quality checks (fmt, vet, lint, test)
make test-coverage # Run tests with coverage report
make security-audit # Run security checks
```

## Usage

**Create a new encrypted vault**

```
sietch init --name dune --encrypt aes256
```

**Add files to the vault**

```
sietch add ./secrets/thumper-plans.pdf
```

**Sync with another vault over LAN**

```
sietch sync --peer 192.168.1.42
```

**Decrypt a file from the vault**

```
sietch decrypt thumper-plans.pdf .
```

**View vault manifest**

```
sietch manifest
```

**Recovery Options**
If your vault becomes corrupted or you need to recover:

```
sietch recover --from .backup
sietch recover --from-remote peer-id
sietch recover --rebuild-metadata
sietch recover --verify-hashes
```

## Roadmap

- Vault initialization and chunk encryption
- LAN peer discovery via UDP broadcast
- TCP file sync with retry and resume
- Optional metadata obfuscation
- WebDAV/SFTP vault mount
- Vault-to-QR export for mobile sync

## Contributing

Sietch is open to contributions - from UX fixes to protocol improvements.

1. Fork this repo
2. Create your feature branch `git checkout -b feature/stillsuit`
3. Commit your changes `git commit -am 'Add stillsuit hydration sync'`
4. Push to the branch `git push origin feature/stillsuit`
5. Create a new Pull Request

## Credits

**Inspiration**

- Syncthing
- IPFS
- Obsidian Sync
- Built with love in Go

## License

This project is licensed under the MIT License - see the LICENSE file for details.

> "When you live in the desert, you develop a very strong survival instinct." - Chani, Dune
