package utils

import (
	commonscrypto "github.com/kamlesh-Sahani/go-commons/crypto"
)

// Hash securely hashes a string (like a password) using Argon2id via go-commons/crypto.
func Hash(data string) (string, error) {
	return commonscrypto.HashArgon2id(data)
}

// VerifyHash verifies that a string matches a given Argon2id encoded hash via go-commons/crypto.
func VerifyHash(data string, encodedHash string) error {
	return commonscrypto.VerifyArgon2id(data, encodedHash)
}

// GenerateRandomPassword generates a cryptographically random secure password.
func GenerateRandomPassword(length int) string {
	return commonscrypto.GenerateRandomPassword(length)
}
