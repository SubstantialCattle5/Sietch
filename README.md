# Sietch Vault

Sietch creates self-contained encrypted vaults that can sync over LAN, sneakernet (USB drives), or weak WiFi networks. It operates fully offline, using chunked data, encryption, and peer-to-peer protocols to ensure your files are always protected and available—even when the internet is not.

---

## Motivation

Sietch Vault is designed for environments where:

* Internet is scarce, censored, or unreliable
* Data privacy is a necessity, not an optional feature
* People work nomadically—researchers, journalists, activists

---

## Core Features

| Feature              | Description                                                           |
| -------------------- | --------------------------------------------------------------------- |
| **AES256/GPG**       | Files are chunked and encrypted with strong symmetric/asymmetric keys |
| **Offline Sync**     | Rsync-style syncing over TCP or LibP2P                                |
| **Gossip Discovery** | Lightweight peer discovery protocol for LAN environments              |
| **CLI First UX**     | Fast, minimal CLI to manage vaults and syncs                          |

---

## CI/CD Status

[![CI](https://github.com/substantialcattle5/sietch/actions/workflows/ci.yml/badge.svg)](https://github.com/substantialcattle5/sietch/actions/workflows/ci.yml)
[![Release](https://github.com/substantialcattle5/sietch/actions/workflows/release.yml/badge.svg)](https://github.com/substantialcattle5/sietch/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/substantialcattle5/sietch/branch/main/graph/badge.svg)](https://codecov.io/gh/substantialcattle5/sietch)
[![Go Report Card](https://goreportcard.com/badge/github.com/substantialcattle5/sietch)](https://goreportcard.com/report/github.com/substantialcattle5/sietch)

---

## How It Works

### Chunking

* Files are split into configurable chunks (default: 4MB)
* Identical chunks across files are deduplicated

### Encryption

Each chunk is encrypted before storage. Options:

* Symmetric passphrase (**AES-256-GCM**)
* Public/private keypairs (**GPG-compatible**)

### Discovery

Peers discover each other via:

* LAN gossip (UDP broadcast)
* Manual IP whitelisting
* (Future) QR-code sharing

### Syncing

Inspired by rsync, Sietch only syncs:

* Missing chunks
* Changed metadata
* Over TCP (with optional compression)

---

## Security Model

| Attack Vector     | Mitigation                               |
| ----------------- | ---------------------------------------- |
| Eavesdropping     | Encrypted chunks over TLS/TCP            |
| Vault tampering   | Merkle trees + hash verification         |
| Metadata leakage  | Optional obfuscation + encrypted indexes |
| Unauthorized sync | Public key signature verification        |

---

## Installation

Releases will provide full install scripts and cross-platform builds.

```bash
git clone https://github.com/SubstantialCattle5/Sietch.git
cd Sietch
go build
```

---

## Development Setup

### Prerequisites

* **Go 1.23** – [Download](https://golang.org/dl/)
* **Node.js 16+** – [Download](https://nodejs.org/)
* **Git** – Version control

### Development Tools (Auto-installed)

* `golangci-lint v1.60.3` – Linting/formatting
* `gosec` – Security scanner
* `Husky` – Git hooks for code quality

### Quick Start

1. **Clone the repo**

   ```bash
   git clone https://github.com/substantialcattle5/sietch.git
   cd sietch
   ```

2. **Run setup**

   ```bash
   ./scripts/setup-hooks.sh
   ```

   This will:

   * Install dependencies
   * Install dev tools (`golangci-lint`, `gosec`)
   * Verify tool versions
   * Set up Git hooks
   * Run initial checks

3. **Verify setup**

   ```bash
   make check-versions
   make build
   make test
   make lint
   ```

### Git Hooks

**Pre-commit:**

* Go formatting (`go fmt`)
* Linting (`golangci-lint`)
* Static analysis (`go vet`)
* Unit tests
* Commit message check (Conventional Commits)

**Pre-push:**

* Full test suite (incl. integration)
* Build verification
* Security audit (`gosec`)

*Bypass hooks:*

```bash
git commit --no-verify -m "skip checks"
git push --no-verify
```

---

## Makefile Commands

```bash
make help            # List commands
make dev             # Format, test, build
make check           # Full quality checks
make test-coverage   # Run tests w/ coverage
make security-audit  # Security checks
```

---

## Usage

**Create a vault**

```bash
sietch init --name dune --encrypt aes256
```

**Add a file**

```bash
sietch add ./secrets/thumper-plans.pdf
```

**Sync over LAN**

```bash
sietch sync --peer 192.168.1.42
```

**Decrypt a file**

```bash
sietch decrypt thumper-plans.pdf .
```

**View manifest**

```bash
sietch manifest
```

**Recovery options**

```bash
sietch recover --from .backup
sietch recover --from-remote peer-id
sietch recover --rebuild-metadata
sietch recover --verify-hashes
```

---


## Contributing

Contributions welcome—from UX polish to protocol improvements.

1. Fork the repo
2. Create a feature branch: `git checkout -b feature/stillsuit`
3. Commit: `git commit -am 'Add stillsuit hydration sync'`
4. Push: `git push origin feature/stillsuit`
5. Open a Pull Request

---

## Credits

**Inspiration**

* Syncthing
* IPFS
* Obsidian Sync
* Built with ❤️ in Go

---

## License

Licensed under the **MIT License** – see the LICENSE file.

> *“When you live in the desert, you develop a very strong survival instinct.”* – Chani, *Dune*
