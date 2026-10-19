package security

import "testing"

func TestGenerateOpaqueTokenIsUniqueAndURLSafe(t *testing.T) {
	tokenA, err := GenerateOpaqueToken()
	if err != nil {
		t.Fatalf("GenerateOpaqueToken returned error: %v", err)
	}
	tokenB, err := GenerateOpaqueToken()
	if err != nil {
		t.Fatalf("GenerateOpaqueToken returned error: %v", err)
	}

	if tokenA == tokenB {
		t.Fatalf("two generated tokens should not collide")
	}

	for _, c := range tokenA {
		if c == '+' || c == '/' || c == '=' {
			t.Fatalf("token contains non-URL-safe character %q", c)
		}
	}
}

func TestHashOpaqueTokenIsDeterministicAndOneWay(t *testing.T) {
	raw := "example-raw-token"

	hash1 := HashOpaqueToken(raw)
	hash2 := HashOpaqueToken(raw)

	if hash1 != hash2 {
		t.Fatalf("hashing the same token twice should produce the same digest")
	}

	if hash1 == raw {
		t.Fatalf("hash must not equal the raw token")
	}

	if HashOpaqueToken("different-token") == hash1 {
		t.Fatalf("different tokens should not collide")
	}
}
