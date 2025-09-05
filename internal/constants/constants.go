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
