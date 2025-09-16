package constants

// vault basic config
const (
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
)

// Encryption types and modes
const (
	EncryptionTypeAES  = "aes"
	EncryptionTypeNone = "none"
	EncryptionTypeGPG  = "gpg"

	AESModeGCM = "gcm"
	AESModeCBC = "cbc"

	KDFScrypt = "scrypt"
	KDFPBKDF2 = "pbkdf2"
)

// File permissions
const (
	SecureDirPerms    = 0o700 // Owner read/write/execute only
	SecureFilePerms   = 0o600 // Owner read/write only
	StandardDirPerms  = 0o755 // Standard directory permissions
	StandardFilePerms = 0o644 // Standard file permissions
)

// Constants for cryptographic and configuration defaults
const (
	// Default KDF parameters
	DefaultScryptN     = 32768 // CPU/memory cost parameter
	DefaultScryptR     = 8     // Block size parameter
	DefaultScryptP     = 1     // Parallelization parameter
	DefaultPBKDF2Iters = 10000 // Default PBKDF2 iteration count

	// RSA key sizes
	DefaultRSAKeySize = 4096 // Default RSA key size for secure operations
	MinRSAKeySize     = 2048 // Minimum acceptable RSA key size

	// Key sizes in bytes
	AESKeySize    = 32 // AES-256 key size
	AESKeySize128 = 16 // AES-128 key size
	AESKeySize192 = 24 // AES-192 key size

	// Nonce and IV sizes
	GCMNonceSize    = 12 // Standard GCM nonce size
	CBCIVSize       = 16 // CBC initialization vector size
	SaltSize        = 16 // Salt size for key derivation
	LegacyNonceSize = 16 // Legacy nonce size for backward compatibility

	// Validation string for key verification
	KeyValidationString = "sietch-key-validation"
)
