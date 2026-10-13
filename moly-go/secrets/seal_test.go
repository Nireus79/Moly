package secrets

import (
	"strings"
	"testing"
)

func TestSealOpenRoundTrip(t *testing.T) {
	secret := "sk-ant-api03-ABCDEFGH"
	sealed, err := Seal("correct horse battery", secret)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(sealed, secret) || !IsSealed(sealed) {
		t.Fatalf("sealed value must not contain the secret: %s", sealed)
	}
	got, err := Open("correct horse battery", sealed)
	if err != nil || got != secret {
		t.Fatalf("Open = %q, %v", got, err)
	}
}

func TestWrongPassphraseAndTamperingAreRefused(t *testing.T) {
	sealed, _ := Seal("correct horse battery", "secret-key")
	if _, err := Open("wrong passphrase!", sealed); err != ErrWrongPassphrase {
		t.Fatalf("wrong passphrase: %v", err)
	}
	tampered := sealed[:len(sealed)-4] + "AAAA"
	if _, err := Open("correct horse battery", tampered); err != ErrWrongPassphrase {
		t.Fatalf("tampered value must not open: %v", err)
	}
	if _, err := Open("correct horse battery", "sealed:v1:!!!not base64"); err != ErrWrongPassphrase {
		t.Fatalf("garbage must not open: %v", err)
	}
}

func TestSealingIsRandomisedAndNeedsAStrongEnoughPassphrase(t *testing.T) {
	a, _ := Seal("correct horse battery", "same")
	b, _ := Seal("correct horse battery", "same")
	if a == b {
		t.Fatal("salt and nonce must differ every time")
	}
	if _, err := Seal("short", "x"); err != ErrShortPass {
		t.Fatalf("short passphrase: %v", err)
	}
}

func TestPlainMarker(t *testing.T) {
	v := Plain("abc")
	if got, ok := FromPlain(v); !ok || got != "abc" || IsSealed(v) {
		t.Fatalf("plain round trip failed: %q %v", got, ok)
	}
	if _, ok := FromPlain("sealed:v1:xxx"); ok {
		t.Fatal("a sealed value is not plain")
	}
}
