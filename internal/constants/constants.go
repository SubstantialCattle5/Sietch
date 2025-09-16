package constants

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
)
