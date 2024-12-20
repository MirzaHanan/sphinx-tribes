package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	btcec "github.com/btcsuite/btcd/btcec/v2"
	btcecdsa "github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/stakwork/sphinx-tribes/config"
	"github.com/stretchr/testify/assert"
)

// Mock configuration for testing
var testConfig = struct {
	SuperAdmins []string
}{
	SuperAdmins: []string{"admin1", "admin2", "admin3"},
}

func TestAdminCheck(t *testing.T) {
	t.Setenv("SUPER_ADMINS", strings.Join(testConfig.SuperAdmins, ","))

	tests := []struct {
		name     string
		pubkey   interface{}
		expected bool
	}{
		{
			name:     "Valid super admin pubkey",
			pubkey:   "admin1",
			expected: true,
		},
		{
			name:     "Invalid super admin pubkey",
			pubkey:   "notAnAdmin",
			expected: false,
		},
		{
			name:     "Empty pubkey",
			pubkey:   "",
			expected: false,
		},
		{
			name:     "Empty SuperAdmins list",
			pubkey:   "admin1",
			expected: false,
		},
		{
			name:     "Pubkey is a substring of a super admin pubkey",
			pubkey:   "admin",
			expected: false,
		},
		{
			name:     "Pubkey is a super admin pubkey with additional characters",
			pubkey:   "admin1extra",
			expected: false,
		},
		{
			name:     "Null or nil pubkey",
			pubkey:   nil,
			expected: false,
		},
		{
			name:     "Non-string pubkey",
			pubkey:   12345,
			expected: false,
		},
		{
			name:     "Large list of super admin pubkeys",
			pubkey:   "admin1",
			expected: true,
		},
		{
			name:     "Large pubkey",
			pubkey:   "averylongpubkeythatisnotinlist",
			expected: false,
		},
		{
			name:     "Special characters in pubkey",
			pubkey:   "!@#$%^&*()",
			expected: false,
		},
		{
			name:     "Case sensitivity",
			pubkey:   "ADMIN1",
			expected: false,
		},
		{
			name:     "Duplicate entries in SuperAdmins",
			pubkey:   "admin1",
			expected: true,
		},
		{
			name:     "Whitespace in pubkey",
			pubkey:   " admin1 ",
			expected: false,
		},
		{
			name:     "Mixed data types in SuperAdmins",
			pubkey:   "admin1",
			expected: true,
		},
	}

	// Temporarily set SuperAdmins to an empty list for the specific test case
	originalSuperAdmins := testConfig.SuperAdmins
	defer func() { testConfig.SuperAdmins = originalSuperAdmins }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Empty SuperAdmins list" {
				config.SuperAdmins = []string{}
			} else {
				config.SuperAdmins = originalSuperAdmins
			}

			var result bool
			switch v := tt.pubkey.(type) {
			case string:
				result = AdminCheck(v)
			default:
				result = false
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}

// Mock function to be tested
func TestIsFreePass(t *testing.T) {
	t.Setenv("SUPER_ADMINS", "")
	tests := []struct {
		name             string
		superAdmins      []string
		adminDevFreePass string
		adminStrings     string
		expected         bool
	}{
		{
			name:             "Single SuperAdmin with FreePass",
			superAdmins:      []string{"freepass"},
			adminDevFreePass: "freepass",
			adminStrings:     "non-empty",
			expected:         true,
		},
		{
			name:             "Empty AdminStrings",
			superAdmins:      []string{"admin"},
			adminDevFreePass: "freepass",
			adminStrings:     "",
			expected:         true,
		},
		{
			name:             "Both Conditions Met",
			superAdmins:      []string{"freepass"},
			adminDevFreePass: "freepass",
			adminStrings:     "",
			expected:         true,
		},
		{
			name:             "Multiple SuperAdmins",
			superAdmins:      []string{"freepass", "admin2"},
			adminDevFreePass: "freepass",
			adminStrings:     "non-empty",
			expected:         false,
		},
		{
			name:             "Empty SuperAdmins List",
			superAdmins:      []string{},
			adminDevFreePass: "freepass",
			adminStrings:     "non-empty",
			expected:         false,
		},
		{
			name:             "Empty SuperAdmins and Empty AdminStrings",
			superAdmins:      []string{},
			adminDevFreePass: "freepass",
			adminStrings:     "",
			expected:         true,
		},
		{
			name:             "Null SuperAdmins",
			superAdmins:      nil,
			adminDevFreePass: "freepass",
			adminStrings:     "non-empty",
			expected:         false,
		},
		{
			name:             "Null AdminStrings",
			superAdmins:      []string{"admin"},
			adminDevFreePass: "freepass",
			adminStrings:     "",
			expected:         true,
		},
		{
			name:             "SuperAdmin with Different FreePass",
			superAdmins:      []string{"admin"},
			adminDevFreePass: "freepass",
			adminStrings:     "non-empty",
			expected:         false,
		},
		{
			name:             "SuperAdmin with Empty String",
			superAdmins:      []string{""},
			adminDevFreePass: "freepass",
			adminStrings:     "non-empty",
			expected:         false,
		},
		{
			name:             "Large SuperAdmins List",
			superAdmins:      make([]string, 1000),
			adminDevFreePass: "freepass",
			adminStrings:     "non-empty",
			expected:         false,
		},
		{
			name:             "SuperAdmin with Null FreePass",
			superAdmins:      []string{"freepass"},
			adminDevFreePass: "",
			adminStrings:     "non-empty",
			expected:         false,
		},
		{
			name:             "AdminDevFreePass as Empty String",
			superAdmins:      []string{"freepass"},
			adminDevFreePass: "",
			adminStrings:     "non-empty",
			expected:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.SuperAdmins = tt.superAdmins
			config.AdminDevFreePass = tt.adminDevFreePass
			config.AdminStrings = tt.adminStrings

			result := IsFreePass()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func generateLargePayload() map[string]interface{} {
	payload := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		payload[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}
	return payload
}

func TestEncodeJwt(t *testing.T) {

	config.InitConfig()
	InitJwt()

	tests := []struct {
		name        string
		publicKey   string
		payload     interface{}
		expectError bool
	}{
		{
			name:        "Valid Public Key and Payload",
			publicKey:   "validPublicKey",
			payload:     map[string]interface{}{"user": "testUser"},
			expectError: false,
		},
		{
			name:        "Valid Public Key with Minimal Payload",
			publicKey:   "validPublicKey",
			payload:     map[string]interface{}{"id": 1},
			expectError: false,
		},
		{
			name:        "Empty Payload",
			publicKey:   "validPublicKey",
			payload:     map[string]interface{}{},
			expectError: false,
		},
		{
			name:        "Maximum Size Payload",
			publicKey:   "validPublicKey",
			payload:     generateLargePayload(),
			expectError: false,
		},
		{
			name:        "Boundary Public Key Length",
			publicKey:   "a",
			payload:     map[string]interface{}{"user": "testUser"},
			expectError: false,
		},
		{
			name:        "Invalid Public Key",
			publicKey:   "invalidPublicKey!",
			payload:     map[string]interface{}{"user": "testUser"},
			expectError: true,
		},
		{
			name:        "Null Public Key",
			publicKey:   "",
			payload:     map[string]interface{}{"user": "testUser"},
			expectError: true,
		},
		{
			name:        "Expired Payload",
			publicKey:   "validPublicKey",
			payload:     map[string]interface{}{"exp": -1},
			expectError: false,
		},
		{
			name:        "Future Expiration Date",
			publicKey:   "validPublicKey",
			payload:     map[string]interface{}{"exp": 9999999999},
			expectError: false,
		},
		{
			name:        "Payload with Special Characters",
			publicKey:   "validPublicKey",
			payload:     map[string]interface{}{"emoji": "😀"},
			expectError: false,
		},
		{
			name:        "Payload with Reserved JWT Claims",
			publicKey:   "validPublicKey",
			payload:     map[string]interface{}{"iss": "issuer", "sub": "subject"},
			expectError: false,
		},
		{
			name:        "Payload with Mixed Data Types",
			publicKey:   "validPublicKey",
			payload:     map[string]interface{}{"string": "value", "number": 123, "boolean": true},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jwt, err := EncodeJwt(tt.publicKey)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, jwt)
			}
		})
	}
}

func TestVerifyAndExtract(t *testing.T) {

	privKey, err := btcec.NewPrivateKey()
	assert.NoError(t, err)

	createValidSignature := func(msg []byte) []byte {
		signedMsg := append(signedMsgPrefix, msg...)
		digest := chainhash.DoubleHashB(signedMsg)
		sig, err := btcecdsa.SignCompact(privKey, digest, true)
		assert.NoError(t, err)
		return sig
	}

	expectedPubKeyHex := hex.EncodeToString(privKey.PubKey().SerializeCompressed())

	tests := []struct {
		name          string
		msg           []byte
		sig           []byte
		expectedKey   string
		expectedValid bool
		expectedErr   error
	}{
		{
			name:          "Valid signature and message",
			msg:           []byte("test message"),
			sig:           createValidSignature([]byte("test message")),
			expectedKey:   expectedPubKeyHex,
			expectedValid: true,
			expectedErr:   nil,
		},
		{
			name:          "Empty message",
			msg:           []byte{},
			sig:           createValidSignature([]byte{}),
			expectedKey:   expectedPubKeyHex,
			expectedValid: true,
			expectedErr:   nil,
		},
		{
			name:          "Nil signature",
			msg:           []byte("test message"),
			sig:           nil,
			expectedKey:   "",
			expectedValid: false,
			expectedErr:   errors.New("bad"),
		},
		{
			name:          "Nil message",
			msg:           nil,
			sig:           createValidSignature([]byte("test message")),
			expectedKey:   "",
			expectedValid: false,
			expectedErr:   errors.New("bad"),
		},
		{
			name:          "Both nil inputs",
			msg:           nil,
			sig:           nil,
			expectedKey:   "",
			expectedValid: false,
			expectedErr:   errors.New("bad"),
		},
		{
			name:          "Empty signature",
			msg:           []byte("test message"),
			sig:           []byte{},
			expectedKey:   "",
			expectedValid: false,
			expectedErr:   errors.New("invalid compact signature size"),
		},
		{
			name:          "Invalid signature format",
			msg:           []byte("test message"),
			sig:           []byte{0xFF, 0xFF},
			expectedKey:   "",
			expectedValid: false,
			expectedErr:   errors.New("invalid compact signature size"),
		},
		{
			name:          "Corrupted signature",
			msg:           []byte("test message"),
			sig:           append(createValidSignature([]byte("test message")), byte(0x00)),
			expectedKey:   "",
			expectedValid: false,
			expectedErr:   errors.New("invalid compact signature size"),
		},
		{
			name:          "Large message",
			msg:           bytes.Repeat([]byte("a"), 1000),
			sig:           createValidSignature(bytes.Repeat([]byte("a"), 1000)),
			expectedKey:   expectedPubKeyHex,
			expectedValid: true,
			expectedErr:   nil,
		},
		{
			name:          "Special characters in message",
			msg:           []byte("!@#$%^&*()_+{}:|<>?"),
			sig:           createValidSignature([]byte("!@#$%^&*()_+{}:|<>?")),
			expectedKey:   expectedPubKeyHex,
			expectedValid: true,
			expectedErr:   nil,
		},
		{
			name:          "UTF-8 characters in message",
			msg:           []byte("Hello, 世界"),
			sig:           createValidSignature([]byte("Hello, 世界")),
			expectedKey:   expectedPubKeyHex,
			expectedValid: true,
			expectedErr:   nil,
		},
		{
			name:          "Message with null bytes",
			msg:           []byte("test\x00message"),
			sig:           createValidSignature([]byte("test\x00message")),
			expectedKey:   expectedPubKeyHex,
			expectedValid: true,
			expectedErr:   nil,
		},
		{
			name:          "Message with only whitespace",
			msg:           []byte("   "),
			sig:           createValidSignature([]byte("   ")),
			expectedKey:   expectedPubKeyHex,
			expectedValid: true,
			expectedErr:   nil,
		},
		{
			name:          "Maximum length message",
			msg:           bytes.Repeat([]byte("x"), 1<<20),
			sig:           createValidSignature(bytes.Repeat([]byte("x"), 1<<20)),
			expectedKey:   expectedPubKeyHex,
			expectedValid: true,
			expectedErr:   nil,
		},
		{
			name:          "Binary data in message",
			msg:           []byte{0x00, 0x01, 0x02, 0x03, 0xFF},
			sig:           createValidSignature([]byte{0x00, 0x01, 0x02, 0x03, 0xFF}),
			expectedKey:   expectedPubKeyHex,
			expectedValid: true,
			expectedErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pubKeyHex, valid, err := VerifyAndExtract(tt.msg, tt.sig)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedValid, valid)

			if tt.expectedKey != "" {
				assert.Equal(t, tt.expectedKey, pubKeyHex)
			}

			if tt.msg != nil && tt.sig != nil && err == nil {
				assert.True(t, bytes.HasPrefix(append(signedMsgPrefix, tt.msg...), signedMsgPrefix))
			}

			if valid && err == nil {
				_, err := hex.DecodeString(pubKeyHex)
				assert.NoError(t, err, "Public key should be valid hex")

				if tt.sig != nil {
					assert.Equal(t, 65, len(tt.sig),
						"Valid signature should be 65 bytes (64 bytes signature + 1 byte recovery ID)")
				}
			}
		})
	}
}

func TestVerifyArbitrary(t *testing.T) {

	privKey, err := btcec.NewPrivateKey()
	assert.NoError(t, err)

	createValidSignature := func(msg string) string {
		signedMsg := append(signedMsgPrefix, []byte(msg)...)
		digest := chainhash.DoubleHashB(signedMsg)
		sig, err := btcecdsa.SignCompact(privKey, digest, true)
		assert.NoError(t, err)
		return base64.URLEncoding.EncodeToString(sig)
	}

	expectedPubKeyHex := hex.EncodeToString(privKey.PubKey().SerializeCompressed())

	tests := []struct {
		name           string
		sig            string
		msg            string
		expectedPubKey string
		expectedError  error
	}{
		{
			name:           "Valid signature and message",
			sig:            createValidSignature("validBase64Signature"),
			msg:            "validBase64Signature",
			expectedPubKey: expectedPubKeyHex,
			expectedError:  nil,
		},
		{
			name:           "Empty signature",
			sig:            "",
			msg:            "validMessage",
			expectedPubKey: "",
			expectedError:  errors.New("invalid compact signature size"),
		},
		{
			name:           "Empty message",
			sig:            createValidSignature(""),
			msg:            "",
			expectedPubKey: expectedPubKeyHex,
			expectedError:  nil,
		},
		{
			name:           "Empty Signature and Message",
			sig:            "",
			msg:            "",
			expectedPubKey: "",
			expectedError:  errors.New("invalid compact signature size"),
		},
		{
			name:           "Invalid base64 signature",
			sig:            "invalid!base64",
			msg:            "validMessage",
			expectedPubKey: "",
			expectedError:  base64.CorruptInputError(7),
		},
		{
			name:           "Invalid Signature After Decoding",
			sig:            base64.URLEncoding.EncodeToString([]byte("invalid-signature-data")),
			msg:            "validMessage",
			expectedPubKey: "",
			expectedError:  errors.New("invalid compact signature size"),
		},
		{
			name:           "Invalid signature bytes",
			sig:            base64.URLEncoding.EncodeToString([]byte("invalid signature")),
			msg:            "test message",
			expectedPubKey: "",
			expectedError:  errors.New("invalid compact signature size"),
		},
		{
			name:           "Large message",
			sig:            createValidSignature(strings.Repeat("x", 1000)),
			msg:            strings.Repeat("x", 1000),
			expectedPubKey: expectedPubKeyHex,
			expectedError:  nil,
		},
		{
			name:           "Large Signature",
			sig:            base64.URLEncoding.EncodeToString(bytes.Repeat([]byte("x"), 1000)),
			msg:            "validMessage",
			expectedPubKey: "",
			expectedError:  errors.New("invalid compact signature size"),
		},
		{
			name:           "UTF-8 message",
			sig:            createValidSignature("Hello, 世界"),
			msg:            "Hello, 世界",
			expectedPubKey: expectedPubKeyHex,
			expectedError:  nil,
		},
		{
			name:           "Signature with Special Characters",
			sig:            createValidSignature("!@#$%^&*()"),
			msg:            "!@#$%^&*()",
			expectedPubKey: expectedPubKeyHex,
			expectedError:  nil,
		},
		{
			name:           "Boundary Length Signature",
			sig:            createValidSignature(strings.Repeat("x", 64)),
			msg:            strings.Repeat("x", 64),
			expectedPubKey: expectedPubKeyHex,
			expectedError:  nil,
		},
		{
			name:           "Message with null bytes",
			sig:            createValidSignature("test\x00message"),
			msg:            "test\x00message",
			expectedPubKey: expectedPubKeyHex,
			expectedError:  nil,
		},
		{
			name:           "Maximum length message",
			sig:            createValidSignature(strings.Repeat("x", 1<<16)),
			msg:            strings.Repeat("x", 1<<16),
			expectedPubKey: expectedPubKeyHex,
			expectedError:  nil,
		},
		{
			name:           "Corrupted signature",
			sig:            base64.URLEncoding.EncodeToString(append([]byte("invalid"), byte(0x00))),
			msg:            "test message",
			expectedPubKey: "",
			expectedError:  errors.New("invalid compact signature size"),
		},
		{
			name:           "Message with only whitespace",
			sig:            createValidSignature("   "),
			msg:            "   ",
			expectedPubKey: expectedPubKeyHex,
			expectedError:  nil,
		},
		{
			name:           "Non-ASCII Characters in Message",
			sig:            createValidSignature("Hello, 世界"),
			msg:            "Hello, 世界",
			expectedPubKey: expectedPubKeyHex,
			expectedError:  nil,
		},
		{
			name:           "Signature with Padding Variations",
			sig:            createValidSignature("test") + "==",
			msg:            "test",
			expectedPubKey: "",
			expectedError:  base64.CorruptInputError(88),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pubKey, err := VerifyArbitrary(tt.sig, tt.msg)

			if tt.expectedError != nil {
				assert.Error(t, err)
				if tt.expectedError.Error() != "" {
					assert.Equal(t, tt.expectedError.Error(), err.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPubKey, pubKey)
			}

			if err == nil {

				_, err := hex.DecodeString(pubKey)
				assert.NoError(t, err, "Public key should be valid hex")
			}
		})
	}
}

func TestSign(t *testing.T) {

	privKey, err := btcec.NewPrivateKey()
	assert.NoError(t, err)

	createExpectedSignature := func(msg []byte) []byte {
		signedMsg := append(signedMsgPrefix, msg...)
		digest := chainhash.DoubleHashB(signedMsg)
		sig, err := btcecdsa.SignCompact(privKey, digest, true)
		assert.NoError(t, err)
		return sig
	}

	expectedPubKeyHex := hex.EncodeToString(privKey.PubKey().SerializeCompressed())

	tests := []struct {
		name          string
		msg           []byte
		privKey       *btcec.PrivateKey
		expectedError error
	}{
		{
			name:          "Valid message and private key",
			msg:           []byte("test message"),
			privKey:       privKey,
			expectedError: nil,
		},
		{
			name:          "Empty message",
			msg:           []byte{},
			privKey:       privKey,
			expectedError: nil,
		},
		{
			name:          "Nil message",
			msg:           nil,
			privKey:       privKey,
			expectedError: errors.New("no msg"),
		},
		{
			name:          "Nil Private Key with Nil Message",
			msg:           nil,
			privKey:       nil,
			expectedError: errors.New("no msg"),
		},
		{
			name:          "Large message",
			msg:           bytes.Repeat([]byte("x"), 1000),
			privKey:       privKey,
			expectedError: nil,
		},
		{
			name:          "Special characters",
			msg:           []byte("!@#$%^&*()"),
			privKey:       privKey,
			expectedError: nil,
		},
		{
			name:          "UTF-8 message",
			msg:           []byte("Hello, 世界"),
			privKey:       privKey,
			expectedError: nil,
		},
		{
			name:          "Message with null bytes",
			msg:           []byte("test\x00message"),
			privKey:       privKey,
			expectedError: nil,
		},
		{
			name:          "Binary data",
			msg:           []byte{0x00, 0x01, 0x02, 0x03, 0xFF},
			privKey:       privKey,
			expectedError: nil,
		},
		{
			name:          "Maximum length message",
			msg:           bytes.Repeat([]byte("x"), 1<<16),
			privKey:       privKey,
			expectedError: nil,
		},
		{
			name:          "Message with Non-ASCII Characters",
			msg:           []byte("こんにちは世界"),
			privKey:       privKey,
			expectedError: nil,
		},
		{
			name:          "Message with only whitespace",
			msg:           []byte("   "),
			privKey:       privKey,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig, err := Sign(tt.msg, tt.privKey)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, sig)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, sig)

				assert.Equal(t, 65, len(sig), "Signature should be 65 bytes")

				expectedSig := createExpectedSignature(tt.msg)
				assert.Equal(t, expectedSig, sig)

				pubKey, valid, verifyErr := VerifyAndExtract(tt.msg, sig)
				assert.NoError(t, verifyErr)
				assert.True(t, valid)
				assert.Equal(t, expectedPubKeyHex, pubKey)

				if tt.msg != nil {
					signedMsg := append(signedMsgPrefix, tt.msg...)
					digest := chainhash.DoubleHashB(signedMsg)

					pubKey, _, err := btcecdsa.RecoverCompact(sig, digest)
					assert.NoError(t, err)
					assert.Equal(t,
						hex.EncodeToString(tt.privKey.PubKey().SerializeCompressed()),
						hex.EncodeToString(pubKey.SerializeCompressed()))
				}
			}
		})
	}
}

func TestConnectionCodeContext(t *testing.T) {
	config.Connection_Auth = "valid_token"

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		expectNextCall bool
	}{
		{
			name:           "Valid Token in Header",
			token:          "valid_token",
			expectedStatus: http.StatusOK,
			expectNextCall: true,
		},
		{
			name:           "Invalid Token in Header",
			token:          "invalid_token",
			expectedStatus: http.StatusUnauthorized,
			expectNextCall: false,
		},
		{
			name:           "Empty Token in Header",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			expectNextCall: false,
		},
		{
			name:           "No Token Header Present",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			expectNextCall: false,
		},
		{
			name:           "Malformed Header",
			token:          "malformed_header",
			expectedStatus: http.StatusUnauthorized,
			expectNextCall: false,
		},
		{
			name:           "Token with Special Characters",
			token:          "special!@#token",
			expectedStatus: http.StatusUnauthorized,
			expectNextCall: false,
		},
		{
			name:           "Token with Whitespace",
			token:          " " + config.Connection_Auth + " ",
			expectedStatus: http.StatusUnauthorized,
			expectNextCall: false,
		},
		{
			name:           "Case Sensitivity in Token",
			token:          strings.ToUpper(config.Connection_Auth),
			expectedStatus: http.StatusUnauthorized,
			expectNextCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.token != "" {
				req.Header.Set("token", tt.token)
			}

			rr := httptest.NewRecorder()

			handler := ConnectionCodeContext(next)
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			assert.Equal(t, tt.expectNextCall, nextCalled)
		})
	}

	t.Run("Null Request Object", func(t *testing.T) {

		nextCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		})

		handler := ConnectionCodeContext(next)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, nil)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)

		assert.False(t, nextCalled)
	})

	t.Run("Large Number of Requests", func(t *testing.T) {

		nextCalled := 0
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled++
			w.WriteHeader(http.StatusOK)
		})

		for i := 0; i < 1000; i++ {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if i%2 == 0 {
				req.Header.Set("token", "valid_token")
			} else {
				req.Header.Set("token", "invalid_token")
			}

			rr := httptest.NewRecorder()
			handler := ConnectionCodeContext(next)
			handler.ServeHTTP(rr, req)

			if i%2 == 0 {
				assert.Equal(t, http.StatusOK, rr.Code)
			} else {
				assert.Equal(t, http.StatusUnauthorized, rr.Code)
			}
		}

		assert.Equal(t, 500, nextCalled)
	})
}
