package commands

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdaterStatementSignsAndVerifies(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "k")
	if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString(priv)), 0o600); err != nil {
		t.Fatal(err)
	}
	m := updaterManifest{Version: "1.2.3", Artifacts: []updaterManifestArtifact{{
		URL: "a", Platform: "linux", Arch: "amd64", DigestAlgo: "sha512", SignatureAlgo: "ed25519ph",
		Digest: base64.StdEncoding.EncodeToString([]byte("0123456789abcdef")),
	}}}
	raw, _ := json.Marshal(m)
	mp := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(mp, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := UpdaterStatement(&UpdaterStatementOptions{Manifest: mp, Key: keyPath}); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(mp)
	var got updaterManifest
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	a := &got.Artifacts[0]
	if err := verifyUpdaterStatement(got.Version, a, pub); err != nil {
		t.Fatal(err)
	}
	a.Platform = "windows"
	if err := verifyUpdaterStatement(got.Version, a, pub); err == nil {
		t.Fatal("a relabelled platform verified")
	}
}
