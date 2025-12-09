// Copyright (C) 2025 Austin Beattie
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package crypto

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = byte(i)
	}

	enc, err := NewEncryptor(key)
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext string
	}{
		{"empty", ""},
		{"short", "abc"},
		{"github pat", "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"},
		{
			"long token",
			"github_pat_11ABCDEFG_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		},
		{"special chars", "token!@#$%^&*()_+-=[]{}|;':\",./<>?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := enc.Encrypt(tt.plaintext)
			require.NoError(t, err)

			if tt.plaintext == "" {
				require.Equal(t, "", encrypted)
				return
			}

			// Encrypted should be different from plaintext
			require.NotEqual(t, tt.plaintext, encrypted)

			// Decrypt should return original
			decrypted, err := enc.Decrypt(encrypted)
			require.NoError(t, err)
			require.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestEncryptProducesDifferentCiphertext(t *testing.T) {
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = byte(i)
	}

	enc, err := NewEncryptor(key)
	require.NoError(t, err)

	plaintext := "ghp_test_token_12345"

	// Encrypt same plaintext twice
	encrypted1, err := enc.Encrypt(plaintext)
	require.NoError(t, err)

	encrypted2, err := enc.Encrypt(plaintext)
	require.NoError(t, err)

	// Should produce different ciphertext due to random nonce
	require.NotEqual(t, encrypted1, encrypted2)

	// Both should decrypt to same plaintext
	decrypted1, err := enc.Decrypt(encrypted1)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted1)

	decrypted2, err := enc.Decrypt(encrypted2)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted2)
}

func TestNewEncryptorInvalidKeySize(t *testing.T) {
	_, err := NewEncryptor([]byte("short"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "key must be 32 bytes")
}

func TestDecryptInvalidBase64(t *testing.T) {
	key := make([]byte, KeySize)
	enc, err := NewEncryptor(key)
	require.NoError(t, err)

	_, err = enc.Decrypt("not-valid-base64!!!")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to decode base64")
}

func TestDecryptTooShort(t *testing.T) {
	key := make([]byte, KeySize)
	enc, err := NewEncryptor(key)
	require.NoError(t, err)

	// Base64 of a very short byte array
	_, err = enc.Decrypt("YWJj") // "abc"
	require.Error(t, err)
	require.Contains(t, err.Error(), "ciphertext too short")
}

func TestLoadOrGenerateKey(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// First call should generate new key
	key1, err := LoadOrGenerateKey(tmpDir)
	require.NoError(t, err)
	require.Len(t, key1, KeySize)

	// Second call should load existing key
	key2, err := LoadOrGenerateKey(tmpDir)
	require.NoError(t, err)
	require.Equal(t, key1, key2)

	// Verify key file exists with correct permissions
	keyFile := filepath.Join(tmpDir, KeyFileName)
	info, err := os.Stat(keyFile)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestMaskToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{"empty", "", ""},
		{"ghp token", "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", "ghp_****xxxx"},
		{"github_pat token", "github_pat_11ABCDEFG_xxxxxxxxxxxxxxxxxxxx", "github_pat_****xxxx"},
		{"short token", "abc", "****"},
		{"medium token", "abcdefgh", "****"},
		{"generic token", "sk_live_xxxxxxxxxxxxxxxxxxxx", "sk_l****xxxx"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskToken(tt.token)
			require.Equal(t, tt.expected, result)
		})
	}
}
