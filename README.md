# Sietch Vault

[![CI](https://github.com/substantialcattle5/sietch/actions/workflows/ci.yml/badge.svg)](https://github.com/substantialcattle5/sietch/actions/workflows/ci.yml)
[![Release](https://github.com/substantialcattle5/sietch/actions/workflows/release.yml/badge.svg)](https://github.com/substantialcattle5/sietch/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/substantialcattle5/sietch/branch/main/graph/badge.svg)](https://codecov.io/gh/substantialcattle5/sietch)
[![Go Report Card](https://goreportcard.com/badge/github.com/substantialcattle5/sietch)](https://goreportcard.com/report/github.com/substantialcattle5/sietch)

Sietch creates self-contained encrypted vaults that can sync over LAN, sneakernet (USB drives), or weak WiFi networks. It operates fully offline, using chunked data, encryption, and peer-to-peer protocols to ensure your files are always protected and available—even when the internet is not.

## Why Sietch?

Sietch Vault is designed for environments where:

* Internet is scarce, censored, or unreliable
* Data privacy is a necessity, not an optional feature
* People work nomadically—researchers, journalists, activists

## Quick Start

### Installation

```bash
git clone https://github.com/substantialcattle5/sietch.git
cd sietch
make build
```

### Basic Usage

**Create a vault**
```bash
sietch init --name dune --key-type aes        # AES-256-GCM encryption
sietch init --name dune --key-type chacha20   # ChaCha20-Poly1305 encryption
```

**Add files**
```bash
# Single file
sietch add ./secrets/thumper-plans.pdf documents/

# Multiple files with individual destinations
sietch add file1.txt dest1/ file2.txt dest2/

# Multiple files to single destination
sietch add ~/photos/img1.jpg ~/photos/img2.jpg vault/photos/
```

**Sync over LAN**
```bash
sietch sync /ip4/192.168.1.42/tcp/4001/p2p/QmPeerID
# or auto-discover peers
sietch sync
```

**Retrieve files**
```bash
sietch get thumper-plans.pdf ./retrieved/
```

## Core Features

| Feature              | Description                                                           |
| -------------------- | --------------------------------------------------------------------- |
| **AES256/GPG**       | Files are chunked and encrypted with strong symmetric/asymmetric keys |
| **ChaCha20**         | Modern authenticated encryption with ChaCha20-Poly1305 AEAD           |
| **Offline Sync**     | Rsync-style syncing over TCP or LibP2P                                |
| **Gossip Discovery** | Lightweight peer discovery protocol for LAN environments              |
| **CLI First UX**     | Fast, minimal CLI to manage vaults and syncs                          |

## How It Works

### Chunking & Deduplication
* Files are split into configurable chunks (default: 4MB)
* Identical chunks across files are deduplicated to save space
* Please Refer [this](internal/deduplication/README.md) documentation to understand how Deduplication works.

### Encryption
Each chunk is encrypted before storage using:
* **Symmetric**: AES-256-GCM or ChaCha20-Poly1305 with passphrase
* **Asymmetric**: GPG-compatible public/private keypairs

### Peer Discovery
Peers discover each other via:
* LAN gossip (UDP broadcast)
* Manual IP whitelisting
* QR-code sharing *(coming soon)*

### Syncing
Inspired by rsync, Sietch only transfers:
* Missing chunks
* Changed metadata
* Over encrypted TCP connections with optional compression

## Available Commands

### Core Operations
```bash
sietch init [flags]                    # Initialize a new vault
sietch add <source> <destination> [args...]  # Add files to vault (multiple file support)
sietch get <filename> <output-path>    # Retrieve files from vault
sietch ls [path]                       # List vault contents
sietch delete <filename>               # Delete files from vault
```

### Network Operations
```bash
sietch discover [flags]                # Discover peers on local network
sietch sync [peer-address]             # Sync with other vaults
sietch sneak [flags]                   # Transfer via sneakernet (USB)
```

### Management
```bash
sietch dedup stats                     # Show deduplication statistics
sietch dedup gc                        # Run garbage collection
sietch dedup optimize                  # Optimize storage
sietch scaffold [flags]                # Create vault from template
```

## Advanced Usage

**View vault contents**
```bash
sietch ls                              # List all files
sietch ls docs/                        # List files in specific directory
sietch ls --long                       # Show detailed information
```

**Network synchronization**
```bash
sietch discover                        # Find peers automatically
sietch sync                            # Auto-discover and sync
sietch sync /ip4/192.168.1.5/tcp/4001/p2p/QmPeerID  # Sync with specific peer
```

**Sneakernet transfer**
```bash
sietch sneak                           # Interactive mode
sietch sneak --source /media/usb/vault # Transfer from USB vault
sietch sneak --dry-run --source /backup/vault  # Preview transfer
```

**Deduplication management**
```bash
sietch dedup stats                     # Show statistics
sietch dedup gc                        # Clean unreferenced chunks
sietch dedup optimize                  # Optimize storage layout
```

## Planned Features (Not Yet Implemented)

The following features are planned for future releases:

```bash
# Recovery operations (planned)
sietch recover --from .backup
sietch recover --from-remote peer-id
sietch recover --rebuild-metadata
sietch recover --verify-hashes

# Standalone decryption (planned)
sietch decrypt <file> <output>

# Direct manifest access (planned)
sietch manifest
```

## Development

### Prerequisites
* **Go 1.23+** – [Download](https://golang.org/dl/)
* **Git** – Version control

### Quick Development Setup

1. **Clone and setup**
    ```bash
    git clone https://github.com/substantialcattle5/sietch.git
    cd sietch
    ./scripts/setup-hooks.sh
    ```

2. **Verify installation**
    ```bash
    make check-versions
    make build
    make test
    ```

### Available Commands
```bash
make help            # List all commands
make dev             # Format, test, build
make check           # Full quality checks
make test-coverage   # Run tests with coverage
make security-audit  # Security checks
```

For detailed development guidelines, see [CONTRIBUTING.md](CONTRIBUTING.md).

## Contributing

We welcome contributions of all kinds! Whether you're fixing bugs, adding features, improving documentation, or enhancing UX.

**Quick contribution steps:**
1. Fork the repository
2. Create a feature branch: `git checkout -b feature/stillsuit`
3. Make your changes following our [style guidelines](CONTRIBUTING.md#styleguides)
4. Commit using [conventional commits](CONTRIBUTING.md#commit-messages)
5. Push and open a Pull Request

See our [Contributing Guide](CONTRIBUTING.md) for detailed information about:
- Development environment setup
- Code style guidelines
- Testing requirements
- Review process

## Inspiration & Credits

Sietch draws inspiration from:
* **Syncthing** - Decentralized file synchronization
* **IPFS** - Content-addressed storage
* **Obsidian Sync** - Seamless cross-device syncing

Built with ❤️ in Go by the open source community.

## Contributors

Thanks to all our amazing contributors!

<!-- readme: contributors -start -->
<table>
	<tbody>
		<tr>
            <td align="center">
                <a href="https://github.com/SubstantialCattle5">
                    <img src="https://avatars.githubusercontent.com/u/92233871?v=4" width="100;" alt="SubstantialCattle5"/>
                    <br />
                    <sub><b>Nilay Nath Sharan</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/MrKeiKun">
                    <img src="https://avatars.githubusercontent.com/u/4362134?v=4" width="100;" alt="MrKeiKun"/>
                    <br />
                    <sub><b>Lorenzo (Kei) Buitizon</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/harshalranjhani">
                    <img src="https://avatars.githubusercontent.com/u/94748669?v=4" width="100;" alt="harshalranjhani"/>
                    <br />
                    <sub><b>Harshal Ranjhani</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/Janmesh23">
                    <img src="https://avatars.githubusercontent.com/u/183159485?v=4" width="100;" alt="Janmesh23"/>
                    <br />
                    <sub><b>Janmesh </b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/ABHINAVGARG05">
                    <img src="https://avatars.githubusercontent.com/u/143117260?v=4" width="100;" alt="ABHINAVGARG05"/>
                    <br />
                    <sub><b>Abhinav Garg</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/Akash29g">
                    <img src="https://avatars.githubusercontent.com/u/77738997?v=4" width="100;" alt="Akash29g"/>
                    <br />
                    <sub><b>Akash Goswami</b></sub>
                </a>
            </td>
		</tr>
		<tr>
            <td align="center">
                <a href="https://github.com/anuja12mishra">
                    <img src="https://avatars.githubusercontent.com/u/109236275?v=4" width="100;" alt="anuja12mishra"/>
                    <br />
                    <sub><b>Anuja Mishra</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/Deepam02">
                    <img src="https://avatars.githubusercontent.com/u/116721751?v=4" width="100;" alt="Deepam02"/>
                    <br />
                    <sub><b>Deepam Goyal</b></sub>
                </a>
            </td>
            <td align="center">
                <a href="https://github.com/Udayan853">
                    <img src="https://avatars.githubusercontent.com/u/76378994?v=4" width="100;" alt="Udayan853"/>
                    <br />
                    <sub><b>Udayan Kulkarni</b></sub>
                </a>
            </td>
		</tr>
	<tbody>
</table>
<!-- readme: contributors -end -->

## License

Licensed under the **MIT License** – see the [LICENSE](LICENSE) file for details.

---

> *"When you live in the desert, you develop a very strong survival instinct."* – Chani, *Dune*
