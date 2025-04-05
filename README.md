# 🏜️ Sietch

**Sietch** is a decentralized, encrypted, portable file system optimized for minimal connectivity 

---

## ✨ Overview

Sietch enables secure, peer-to-peer file syncing and storage across unreliable or disconnected networks. Think of it as an **IPFS-lite**, **Syncthing-inspired** system — but built for digital survival in harsh environments.

- 🔐 End-to-end encrypted file chunks
- 📦 Deduplicated chunk-based storage
- 🌍 Peer discovery via LAN or static IPs
- 🔄 Sync files between machines with minimal bandwidth
- 💻 Simple CLI interface for creating and syncing vaults
- 🧱 Offline-first, portable, and durable

---

## ⚙️ Use Cases

- Operators in remote areas or low-connectivity zones
- Secure, encrypted backups over LAN or sneakernet (USB sticks)
- Field researchers syncing encrypted data
- Nomadic workspaces with ephemeral storage

---

## 🚀 Features

| Feature               | Description                                                 |
|----------------------|-------------------------------------------------------------|
| 🔐 AES256/GPG Support | Chunk-level encryption using symmetric or asymmetric keys  |
| 📦 Content-Addressed  | Every file is chunked and stored by hash (Merkle-DAG)      |
| 🌐 Peer Syncing       | Lightweight P2P syncing via LibP2P or TCP                  |
| 🔄 Incremental Uploads| Rsync-style syncs for large files and low bandwidth        |
| 📁 Mountable Vaults   | Local or remote Sietch storage (WebDAV, USB, etc.)         |
| 💻 CLI-First UX       | Fast, scriptable CLI interface                             |

---

## 📦 Installation

```bash 

> sietch status
🟢 Local node: Arrakis
🧱 Chunks stored: 1,254
🔐 Vault: Encrypted (AES-256)
🌍 Known peers: 3 (Caladan, GiediPrime, Salusa)

> sietch sync
[+] Found peer 'Caladan' on 192.168.1.4
[+] Syncing... 12 new chunks downloaded.
✅ Sync complete. All data up to date.


```


---

💬 Philosophy

This project is built on the ideas of:

Resilience over convenience

Privacy without compromise

Portability for any terrain

If the world goes offline, your data should still be safe.

---

> "_The mystery of life isn't a problem to solve, but a reality to experience._"  
> — Frank Herbert, *Dune*





