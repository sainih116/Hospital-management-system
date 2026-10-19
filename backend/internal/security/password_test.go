package security

import "testing"

func TestHashPasswordAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("Sup3rSecret!")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if hash == "Sup3rSecret!" {
		t.Fatalf("password hash must not equal the plaintext password")
	}

	if !VerifyPassword(hash, "Sup3rSecret!") {
		t.Fatalf("VerifyPassword should accept the correct password")
	}

	if VerifyPassword(hash, "wrong-password") {
		t.Fatalf("VerifyPassword should reject an incorrect password")
	}
}

func TestHashPasswordProducesUniqueSalts(t *testing.T) {
	hashA, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	hashB, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if hashA == hashB {
		t.Fatalf("two hashes of the same password should differ (unique bcrypt salt)")
	}
}
