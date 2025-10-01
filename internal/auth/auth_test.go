package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHashPassword(t *testing.T) {
	password := "mySecurePassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	if hash == password {
		t.Error("Hash should not equal plain password")
	}
}

func TestCheckPasswordHash_ValidPassword(t *testing.T) {
	password := "mySecurePassword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	match, err := CheckPasswordHash(password, hash)
	if err != nil {
		t.Fatalf("CheckPasswordHash failed: %v", err)
	}

	if !match {
		t.Error("Expected password to match hash")
	}
}

func TestCheckPasswordHash_InvalidPassword(t *testing.T) {
	password := "mySecurePassword123"
	wrongPassword := "wrongPassword"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	match, err := CheckPasswordHash(wrongPassword, hash)
	if err != nil {
		t.Fatalf("CheckPasswordHash failed: %v", err)
	}

	if match {
		t.Error("Expected password to NOT match hash")
	}
}

func TestMakeJWT(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret-key"
	expiresIn := time.Hour

	token, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT failed: %v", err)
	}

	if token == "" {
		t.Error("Token should not be empty")
	}
}

func TestValidateJWT_ValidToken(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret-key"
	expiresIn := time.Hour

	// Create token
	token, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Validate token
	extractedID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("ValidateJWT failed: %v", err)
	}

	if extractedID != userID {
		t.Errorf("Expected user ID %v, got %v", userID, extractedID)
	}
}

func TestValidateJWT_WrongSecret(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret-key"
	wrongSecret := "wrong-secret-key"
	expiresIn := time.Hour

	// Create token with correct secret
	token, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Try to validate with wrong secret
	_, err = ValidateJWT(token, wrongSecret)
	if err == nil {
		t.Error("Expected error when validating with wrong secret")
	}
}

func TestValidateJWT_ExpiredToken(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret-key"
	expiresIn := -time.Hour // Already expired

	// Create expired token
	token, err := MakeJWT(userID, secret, expiresIn)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Try to validate expired token
	_, err = ValidateJWT(token, secret)
	if err == nil {
		t.Error("Expected error when validating expired token")
	}
}

func TestValidateJWT_InvalidFormat(t *testing.T) {
	secret := "test-secret-key"
	invalidToken := "not.a.valid.jwt"

	_, err := ValidateJWT(invalidToken, secret)
	if err == nil {
		t.Error("Expected error when validating invalid token format")
	}
}

func TestValidateJWT_EmptyToken(t *testing.T) {
	secret := "test-secret-key"

	_, err := ValidateJWT("", secret)
	if err == nil {
		t.Error("Expected error when validating empty token")
	}
}

func TestHashPassword_DifferentHashesForSamePassword(t *testing.T) {
	password := "samePassword"

	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("First hash failed: %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Second hash failed: %v", err)
	}

	// Hashes should be different due to random salt
	if hash1 == hash2 {
		t.Error("Expected different hashes for same password (salt should differ)")
	}

	// But both should validate correctly
	match1, _ := CheckPasswordHash(password, hash1)
	match2, _ := CheckPasswordHash(password, hash2)

	if !match1 || !match2 {
		t.Error("Both hashes should validate correctly")
	}
}
