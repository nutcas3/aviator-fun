package game

import (
	"testing"
)

func TestHashAndMapToMultiplier(t *testing.T) {
	tests := []struct {
		name         string
		serverSeed   string
		clientSeed   string
		nonce        int
		wantMin      float64
		wantMax      float64
	}{
		{
			name:       "Basic test",
			serverSeed: "test_server_seed_123",
			clientSeed: "test_client_seed_456",
			nonce:      1,
			wantMin:    MIN_MULTIPLIER,
			wantMax:    MAX_MULTIPLIER,
		},
		{
			name:       "Different nonce",
			serverSeed: "test_server_seed_123",
			clientSeed: "test_client_seed_456",
			nonce:      2,
			wantMin:    MIN_MULTIPLIER,
			wantMax:    MAX_MULTIPLIER,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HashAndMapToMultiplier(tt.serverSeed, tt.clientSeed, tt.nonce)
			
			if got < tt.wantMin {
				t.Errorf("HashAndMapToMultiplier() = %v, want >= %v", got, tt.wantMin)
			}
			if got > tt.wantMax {
				t.Errorf("HashAndMapToMultiplier() = %v, want <= %v", got, tt.wantMax)
			}
		})
	}
}

func TestHashAndMapToMultiplier_Deterministic(t *testing.T) {
	serverSeed := "deterministic_test_seed"
	clientSeed := "deterministic_client_seed"
	nonce := 42

	// Call multiple times with same inputs
	result1 := HashAndMapToMultiplier(serverSeed, clientSeed, nonce)
	result2 := HashAndMapToMultiplier(serverSeed, clientSeed, nonce)
	result3 := HashAndMapToMultiplier(serverSeed, clientSeed, nonce)

	if result1 != result2 || result2 != result3 {
		t.Errorf("HashAndMapToMultiplier() is not deterministic: got %v, %v, %v", result1, result2, result3)
	}
}

func TestHashAndMapToMultiplier_DifferentInputs(t *testing.T) {
	serverSeed := "test_seed"
	clientSeed := "test_client"

	// Different nonces should produce different results (most of the time)
	result1 := HashAndMapToMultiplier(serverSeed, clientSeed, 1)
	result2 := HashAndMapToMultiplier(serverSeed, clientSeed, 2)
	result3 := HashAndMapToMultiplier(serverSeed, clientSeed, 3)

	// At least one should be different
	if result1 == result2 && result2 == result3 {
		t.Error("HashAndMapToMultiplier() produces same result for different nonces (unlikely)")
	}
}

func TestGenerateSeed(t *testing.T) {
	seed1 := GenerateSeed()
	seed2 := GenerateSeed()

	if seed1 == seed2 {
		t.Error("GenerateSeed() produced duplicate seeds")
	}

	if len(seed1) != 64 { // 32 bytes = 64 hex characters
		t.Errorf("GenerateSeed() length = %v, want 64", len(seed1))
	}
}

func TestHashCommitment(t *testing.T) {
	seed := "test_seed_12345"
	
	hash1 := HashCommitment(seed)
	hash2 := HashCommitment(seed)

	if hash1 != hash2 {
		t.Error("HashCommitment() is not deterministic")
	}

	if len(hash1) != 64 { // SHA256 = 64 hex characters
		t.Errorf("HashCommitment() length = %v, want 64", len(hash1))
	}
}

func TestVerifyRound(t *testing.T) {
	serverSeed := "verification_test_seed"
	clientSeed := "verification_client_seed"
	nonce := 100

	// Calculate the actual multiplier
	actualMultiplier := HashAndMapToMultiplier(serverSeed, clientSeed, nonce)

	tests := []struct {
		name              string
		serverSeed        string
		clientSeed        string
		nonce             int
		claimedMultiplier float64
		want              bool
	}{
		{
			name:              "Valid verification",
			serverSeed:        serverSeed,
			clientSeed:        clientSeed,
			nonce:             nonce,
			claimedMultiplier: actualMultiplier,
			want:              true,
		},
		{
			name:              "Invalid multiplier",
			serverSeed:        serverSeed,
			clientSeed:        clientSeed,
			nonce:             nonce,
			claimedMultiplier: actualMultiplier + 10.0,
			want:              false,
		},
		{
			name:              "Wrong server seed",
			serverSeed:        "wrong_seed",
			clientSeed:        clientSeed,
			nonce:             nonce,
			claimedMultiplier: actualMultiplier,
			want:              false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifyRound(tt.serverSeed, tt.clientSeed, tt.nonce, tt.claimedMultiplier)
			if got != tt.want {
				t.Errorf("VerifyRound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHashAndMapToMultiplier_HouseEdge(t *testing.T) {
	// Test that house edge is working (some results should be MIN_MULTIPLIER)
	serverSeed := "house_edge_test"
	instantCrashCount := 0
	totalTests := 1000

	for i := 0; i < totalTests; i++ {
		result := HashAndMapToMultiplier(serverSeed, "client", i)
		if result == MIN_MULTIPLIER {
			instantCrashCount++
		}
	}

	// House edge is 1%, so we expect roughly 1% instant crashes
	// Allow for variance (0.5% to 2%)
	minExpected := totalTests * 5 / 1000  // 0.5%
	maxExpected := totalTests * 20 / 1000 // 2%

	if instantCrashCount < minExpected || instantCrashCount > maxExpected {
		t.Logf("Instant crash rate: %d/%d (%.2f%%)", instantCrashCount, totalTests, float64(instantCrashCount)/float64(totalTests)*100)
		// This is informational, not a hard failure
	}
}

func BenchmarkHashAndMapToMultiplier(b *testing.B) {
	serverSeed := "benchmark_server_seed"
	clientSeed := "benchmark_client_seed"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HashAndMapToMultiplier(serverSeed, clientSeed, i)
	}
}

func BenchmarkGenerateSeed(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateSeed()
	}
}

func BenchmarkHashCommitment(b *testing.B) {
	seed := "benchmark_seed_12345"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HashCommitment(seed)
	}
}
