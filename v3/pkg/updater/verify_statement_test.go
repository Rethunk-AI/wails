package updater

import (
	"crypto/ed25519"
	"crypto/sha512"
	"testing"
)

func statementFixture(t *testing.T) (cfg *Config, rel *Release, digest []byte, priv ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha512.Sum512([]byte("payload"))
	digest = sum[:]
	cfg = &Config{PublicKey: pub, Platform: "linux", Arch: "amd64", CurrentVersion: "1.0.0", RequireStatement: true}
	rel = &Release{Version: "1.1.0", Verification: &Verification{DigestAlgo: "sha512", SignatureAlgo: "ed25519ph", Digest: digest}}
	rel.Verification.StatementSignature = ed25519.Sign(priv, Statement("1.1.0", "linux", "amd64", "sha512", digest))
	return cfg, rel, digest, priv
}

func TestVerifyStatementAcceptsTheSignedRelease(t *testing.T) {
	cfg, rel, digest, _ := statementFixture(t)
	if err := verifyStatement(rel, cfg, digest); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyStatementRefusesDowngradeAndRelabel(t *testing.T) {
	cfg, rel, digest, priv := statementFixture(t)
	// An older signed build relabelled as newer: the signature covers the old version.
	old := *rel
	old.Verification = &Verification{DigestAlgo: "sha512", SignatureAlgo: "ed25519ph",
		StatementSignature: ed25519.Sign(priv, Statement("0.9.0", "linux", "amd64", "sha512", digest))}
	if err := verifyStatement(&old, cfg, digest); err == nil {
		t.Fatal("a signature for 0.9.0 verified as 1.1.0")
	}
	// A genuinely signed release that is not newer than the running one.
	cfg.CurrentVersion = "1.1.0"
	if err := verifyStatement(rel, cfg, digest); err == nil {
		t.Fatal("a release equal to the running version was accepted")
	}
	cfg.CurrentVersion = "2.0.0"
	if err := verifyStatement(rel, cfg, digest); err == nil {
		t.Fatal("a downgrade was accepted")
	}
}

func TestVerifyStatementRefusesAnotherPlatformsFile(t *testing.T) {
	cfg, rel, digest, _ := statementFixture(t)
	cfg.Platform = "windows"
	if err := verifyStatement(rel, cfg, digest); err == nil {
		t.Fatal("a linux statement verified for windows")
	}
	cfg.Platform, cfg.Arch = "linux", "arm64"
	if err := verifyStatement(rel, cfg, digest); err == nil {
		t.Fatal("an amd64 statement verified for arm64")
	}
	cfg.Arch = "amd64"
	other := sha512.Sum512([]byte("another file"))
	if err := verifyStatement(rel, cfg, other[:]); err == nil {
		t.Fatal("a statement verified for a different file")
	}
}

func TestVerifyStatementRefusesAnOldFormatRelease(t *testing.T) {
	cfg, rel, digest, _ := statementFixture(t)
	rel.Verification.StatementSignature = nil
	if err := verifyStatement(rel, cfg, digest); err == nil {
		t.Fatal("a release with no statement installed under RequireStatement")
	}
}
