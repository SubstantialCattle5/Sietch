package constants

const (
	//** Vault basic config

	// Vault name
	VaultNameLabel     = "Vault name"
	VaultNameDefault   = "my-sietch"
	VaultNameMinLength = 3

	// author config
	AuthorLabel     = "Author"
	AuthorDefault   = "nilay@dune.net"
	AuthorMinLength = 3
	AuthorAllowEdit = true

	// tags config
	TagsLabel     = "Tags (comma-separated)"
	TagsDefault   = "research,desert,offline"
	TagsAllowEdit = true

	// Encryption types and modes
	EncryptionTypeAES      = "aes"
	EncryptionTypeNone     = "none"
	EncryptionTypeGPG      = "gpg"
	EncryptionTypeChaCha20 = "chacha20"

	AESModeGCM = "gcm"
	AESModeCBC = "cbc"

	KDFScrypt = "scrypt"
	KDFPBKDF2 = "pbkdf2"

	//** File permissions

	SecureDirPerms    = 0o700 // Owner read/write/execute only
	SecureFilePerms   = 0o600 // Owner read/write only
	StandardDirPerms  = 0o755 // Standard directory permissions
	StandardFilePerms = 0o644 // Standard file permissions

	//** Constants for cryptographic and configuration defaults

	// Default KDF parameters
	DefaultScryptN     = 32768 // CPU/memory cost parameter
	DefaultScryptR     = 8     // Block size parameter
	DefaultScryptP     = 1     // Parallelization parameter
	DefaultPBKDF2Iters = 10000 // Default PBKDF2 iteration count

	// RSA key sizes
	DefaultRSAKeySize = 4096 // Default RSA key size for secure operations
	MinRSAKeySize     = 2048 // Minimum acceptable RSA key size
	Ed25519KeySize    = 256  // Ed25519 key size

	// Key sizes in bytes
	AESKeySize    = 32 // AES-256 key size
	AESKeySize128 = 16 // AES-128 key size
	AESKeySize192 = 24 // AES-192 key size

	// Nonce and IV sizes
	GCMNonceSize    = 12 // Standard GCM nonce size
	CBCIVSize       = 16 // CBC initialization vector size
	SaltSize        = 16 // Salt size for key derivation
	LegacyNonceSize = 16 // Legacy nonce size for backward compatibility

	// GPG key types
	GPGKeyTypeRSA     = "rsa"
	GPGKeyTypeEd25519 = "ed25519"

	// GPG key expiration
	GPGKeyExpiration1Year  = "1y"
	GPGKeyExpiration2Years = "2y"
	GPGKeyExpiration5Years = "5y"
	GPGKeyExpirationNever  = "0"

	// Validation string for key verification
	KeyValidationString = "sietch-key-validation"

	//** Constants for chunking

	DefaultChunkSize = 4 * 1024 * 1024 // 4MB

	//** Constants for compression
	CompressionTypeGzip = "gzip"
	CompressionTypeZstd = "zstd"
	CompressionTypeNone = "none"
	CompressionTypeLZ4  = "lz4"


	// Maximum decompression size to prevent decompression bombs
	// This should be large enough for legitimate chunks but prevent DoS attacks
	MaxDecompressionSize = 100 * 1024 * 1024 // 100MB max decompressed size

	//** Constants for hash algorithms
	HashAlgorithmSHA256 = "sha256"
	HashAlgorithmSHA512 = "sha512"
	HashAlgorithmSHA1   = "sha1"
	HashAlgorithmBLAKE3 = "blake3"

	//* Regex
	EmailRegex = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
)
