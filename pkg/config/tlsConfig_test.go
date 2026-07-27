// Copyright Contributors to the Open Cluster Management project

package config

import (
	"crypto/tls"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseTLSVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected uint16
	}{
		{"VersionTLS10", tls.VersionTLS10},
		{"VersionTLS11", tls.VersionTLS11},
		{"VersionTLS12", tls.VersionTLS12},
		{"VersionTLS13", tls.VersionTLS13},
		{"", tls.VersionTLS12},               // empty defaults to 1.2
		{"InvalidVersion", tls.VersionTLS12}, // unknown defaults to 1.2
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseTLSVersion(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCipherSuitesFromNames(t *testing.T) {
	t.Run("valid IANA ciphers", func(t *testing.T) {
		result := cipherSuitesFromNames("TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384")
		assert.Len(t, result, 2)
		assert.Equal(t, tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, result[0])
		assert.Equal(t, tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384, result[1])
	})

	t.Run("empty string", func(t *testing.T) {
		result := cipherSuitesFromNames("")
		assert.Nil(t, result)
	})

	t.Run("unknown ciphers skipped", func(t *testing.T) {
		result := cipherSuitesFromNames("TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,UNKNOWN_CIPHER")
		assert.Len(t, result, 1)
		assert.Equal(t, tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, result[0])
	})

	t.Run("whitespace trimmed", func(t *testing.T) {
		result := cipherSuitesFromNames(" TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 , TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384 ")
		assert.Len(t, result, 2)
	})

	t.Run("TLS 1.3 ciphers recognized", func(t *testing.T) {
		result := cipherSuitesFromNames("TLS_AES_128_GCM_SHA256,TLS_AES_256_GCM_SHA384,TLS_CHACHA20_POLY1305_SHA256")
		assert.Len(t, result, 3)
	})
}

func TestTlsConfigFromConfigMap(t *testing.T) {
	t.Run("Intermediate profile", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "ocm-tls-profile"},
			Data: map[string]string{
				"profileType":   "Intermediate",
				"minTLSVersion": "VersionTLS12",
				"cipherSuites":  "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			},
		}
		cfg := tlsConfigFromConfigMap(cm)
		assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
		assert.Len(t, cfg.CipherSuites, 2)
	})

	t.Run("Modern profile", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "ocm-tls-profile"},
			Data: map[string]string{
				"profileType":   "Modern",
				"minTLSVersion": "VersionTLS13",
				"cipherSuites":  "TLS_AES_128_GCM_SHA256,TLS_AES_256_GCM_SHA384,TLS_CHACHA20_POLY1305_SHA256",
			},
		}
		cfg := tlsConfigFromConfigMap(cm)
		assert.Equal(t, uint16(tls.VersionTLS13), cfg.MinVersion)
		assert.Len(t, cfg.CipherSuites, 3)
	})

	t.Run("Old profile", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "ocm-tls-profile"},
			Data: map[string]string{
				"profileType":   "Old",
				"minTLSVersion": "VersionTLS10",
				"cipherSuites":  "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			},
		}
		cfg := tlsConfigFromConfigMap(cm)
		assert.Equal(t, uint16(tls.VersionTLS10), cfg.MinVersion)
		assert.Len(t, cfg.CipherSuites, 1)
	})

	t.Run("Intermediate fallback", func(t *testing.T) {
		cfg := intermediateProfileTLSConfig()
		assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
		assert.Greater(t, len(cfg.CipherSuites), 0, "should have cipher suites from Intermediate profile")
	})

	t.Run("empty ciphers uses Go defaults", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "ocm-tls-profile"},
			Data: map[string]string{
				"profileType":   "Intermediate",
				"minTLSVersion": "VersionTLS12",
				"cipherSuites":  "",
			},
		}
		cfg := tlsConfigFromConfigMap(cm)
		assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
		assert.Nil(t, cfg.CipherSuites)
	})

	t.Run("missing minTLSVersion key defaults to TLS 1.2", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "ocm-tls-profile"},
			Data: map[string]string{
				"profileType":  "Intermediate",
				"cipherSuites": "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			},
		}
		cfg := tlsConfigFromConfigMap(cm)
		assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
	})
}

func TestCipherSuitesFromNames_AllUnknown(t *testing.T) {
	result := cipherSuitesFromNames("FAKE_CIPHER_1,FAKE_CIPHER_2")
	assert.Empty(t, result, "should return no valid cipher suites")
}

func TestIntermediateProfileTLSConfig_MatchesOpenShiftProfile(t *testing.T) {
	cfg := intermediateProfileTLSConfig()
	profile := configv1.TLSProfiles[configv1.TLSProfileIntermediateType]

	assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
	// Should have resolved at least some of the profile's ciphers.
	// The exact count may differ if Go doesn't support all OpenSSL ciphers.
	assert.Greater(t, len(cfg.CipherSuites), 0)
	assert.LessOrEqual(t, len(cfg.CipherSuites), len(profile.Ciphers),
		"should not have more ciphers than the profile defines")
}

func TestGetTLSConfigFromEnv(t *testing.T) {
	t.Run("both env vars set", func(t *testing.T) {
		t.Setenv("TLS_MIN_VERSION", "772") // TLS 1.3
		t.Setenv("TLS_CIPHERS", "TLS_AES_128_GCM_SHA256,TLS_AES_256_GCM_SHA384")

		cfg := getTLSConfigFromEnv()
		assert.Equal(t, uint16(tls.VersionTLS13), cfg.MinVersion)
		assert.Len(t, cfg.CipherSuites, 2)
	})

	t.Run("only TLS_MIN_VERSION set", func(t *testing.T) {
		t.Setenv("TLS_MIN_VERSION", "771") // TLS 1.2
		t.Setenv("TLS_CIPHERS", "")

		cfg := getTLSConfigFromEnv()
		assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
		assert.Nil(t, cfg.CipherSuites)
	})

	t.Run("invalid TLS_MIN_VERSION falls back to TLS 1.2", func(t *testing.T) {
		t.Setenv("TLS_MIN_VERSION", "not-a-number")
		t.Setenv("TLS_CIPHERS", "")

		cfg := getTLSConfigFromEnv()
		assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
	})

	t.Run("no env vars falls back to Intermediate profile", func(t *testing.T) {
		t.Setenv("TLS_MIN_VERSION", "")
		t.Setenv("TLS_CIPHERS", "")

		cfg := getTLSConfigFromEnv()
		assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
		assert.Greater(t, len(cfg.CipherSuites), 0, "Intermediate profile should have cipher suites")
	})

	t.Run("all unknown ciphers uses Go defaults", func(t *testing.T) {
		t.Setenv("TLS_MIN_VERSION", "771")
		t.Setenv("TLS_CIPHERS", "FAKE_CIPHER_1,FAKE_CIPHER_2")

		cfg := getTLSConfigFromEnv()
		assert.Nil(t, cfg.CipherSuites, "should fall back to nil (Go defaults) when no ciphers resolve")
	})
}
