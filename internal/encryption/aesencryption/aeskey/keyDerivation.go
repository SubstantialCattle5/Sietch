package aeskey

import (
	"crypto/sha256"
	"fmt"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
)

// KDFConfig holds parameters for key derivation
type KDFConfig struct {
	Algorithm string
	Salt      []byte
	// Scrypt parameters
	ScryptN int
	ScryptR int
	ScryptP int
	// PBKDF2 parameters
	PBKDF2Iterations int
}

// DeriveKey derives a key from a passphrase using the specified KDF algorithm
func DeriveKey(passphrase string, config KDFConfig) ([]byte, error) {
	switch config.Algorithm {
	case constants.KDFScrypt:
		return deriveScryptKey(passphrase, config)
	case constants.KDFPBKDF2:
		return derivePBKDF2Key(passphrase, config)
	default:
		return nil, fmt.Errorf("unsupported KDF algorithm: %s", config.Algorithm)
	}
}

// deriveScryptKey derives a key using the scrypt algorithm
func deriveScryptKey(passphrase string, config KDFConfig) ([]byte, error) {
	return scrypt.Key(
		[]byte(passphrase),
		config.Salt,
		config.ScryptN,
		config.ScryptR,
		config.ScryptP,
		constants.AESKeySize, // 32 bytes for AES-256
	)
}

// derivePBKDF2Key derives a key using the PBKDF2 algorithm
func derivePBKDF2Key(passphrase string, config KDFConfig) ([]byte, error) {
	return pbkdf2.Key(
		[]byte(passphrase),
		config.Salt,
		config.PBKDF2Iterations,
		constants.AESKeySize, // 32 bytes for AES-256
		sha256.New,
	), nil
}

// SetupKDFDefaults applies default KDF parameters to the vault configuration
func SetupKDFDefaults(cfg *config.VaultConfig) {
	if cfg.Encryption.AESConfig.KDF == "" {
		cfg.Encryption.AESConfig.KDF = constants.KDFScrypt
	}

	switch cfg.Encryption.AESConfig.KDF {
	case constants.KDFScrypt:
		setupScryptDefaults(cfg)
	case constants.KDFPBKDF2:
		setupPBKDF2Defaults(cfg)
	}
}

// setupScryptDefaults sets default scrypt parameters if not already configured
func setupScryptDefaults(cfg *config.VaultConfig) {
	if cfg.Encryption.AESConfig.ScryptN == 0 {
		cfg.Encryption.AESConfig.ScryptN = constants.DefaultScryptN
	}
	if cfg.Encryption.AESConfig.ScryptR == 0 {
		cfg.Encryption.AESConfig.ScryptR = constants.DefaultScryptR
	}
	if cfg.Encryption.AESConfig.ScryptP == 0 {
		cfg.Encryption.AESConfig.ScryptP = constants.DefaultScryptP
	}
}

// setupPBKDF2Defaults sets default PBKDF2 parameters if not already configured
func setupPBKDF2Defaults(cfg *config.VaultConfig) {
	if cfg.Encryption.AESConfig.PBKDF2I == 0 {
		cfg.Encryption.AESConfig.PBKDF2I = constants.DefaultPBKDF2Iters
	}
}

// BuildKDFConfig creates a KDFConfig from vault configuration
func BuildKDFConfig(cfg *config.VaultConfig, salt []byte) KDFConfig {
	return KDFConfig{
		Algorithm:        cfg.Encryption.AESConfig.KDF,
		Salt:             salt,
		ScryptN:          cfg.Encryption.AESConfig.ScryptN,
		ScryptR:          cfg.Encryption.AESConfig.ScryptR,
		ScryptP:          cfg.Encryption.AESConfig.ScryptP,
		PBKDF2Iterations: cfg.Encryption.AESConfig.PBKDF2I,
	}
}

// CopyKDFParametersToKeyConfig copies KDF parameters from vault config to key config
func CopyKDFParametersToKeyConfig(vaultCfg *config.VaultConfig, keyCfg *config.KeyConfig) {
	keyCfg.AESConfig.KDF = vaultCfg.Encryption.AESConfig.KDF

	switch vaultCfg.Encryption.AESConfig.KDF {
	case constants.KDFScrypt:
		keyCfg.AESConfig.ScryptN = vaultCfg.Encryption.AESConfig.ScryptN
		keyCfg.AESConfig.ScryptR = vaultCfg.Encryption.AESConfig.ScryptR
		keyCfg.AESConfig.ScryptP = vaultCfg.Encryption.AESConfig.ScryptP
	case constants.KDFPBKDF2:
		keyCfg.AESConfig.PBKDF2I = vaultCfg.Encryption.AESConfig.PBKDF2I
	}
}
