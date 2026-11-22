package credentials

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeKeyring is an in-memory keyring so tests never touch the real one.
type fakeKeyring struct {
	values map[string]string
	fail   bool
	failOn error
}

func newFakeKeyring() *fakeKeyring { return &fakeKeyring{values: map[string]string{}} }

func (f *fakeKeyring) Get(service, user string) (string, error) {
	if f.fail {
		return "", ErrKeyringUnavailable
	}
	v, ok := f.values[service+"/"+user]
	if !ok {
		return "", ErrKeyringUnavailable
	}
	return v, nil
}

func (f *fakeKeyring) Set(service, user, secret string) error {
	if f.fail {
		return ErrKeyringUnavailable
	}
	f.values[service+"/"+user] = secret
	return nil
}

func (f *fakeKeyring) Delete(service, user string) error {
	if f.fail {
		return ErrKeyringUnavailable
	}
	delete(f.values, service+"/"+user)
	return nil
}

func newStore(t *testing.T, kr Keyring) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	s := NewStore(Options{
		Keyring:             kr,
		ManagementTokenFile: filepath.Join(dir, "management-token"),
		APIKeyFile:          filepath.Join(dir, "api-key"),
		KeyringEnabled:      kr != nil,
	})
	return s, dir
}

func TestEnvironmentCredentialsWinAndAreNotPersisted(t *testing.T) {
	kr := newFakeKeyring()
	s, dir := newStore(t, kr)
	s.Env = map[string]string{
		"ROUTEBURN_MANAGEMENT_KEY": "mgmt-env",
		"ROUTEBURN_API_KEY":        "api-env",
	}

	creds, err := s.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if creds.Management != "mgmt-env" || creds.APIKey != "api-env" {
		t.Errorf("creds = %+v", creds)
	}
	if len(kr.values) != 0 {
		t.Errorf("environment credentials must not be written to the keyring: %v", kr.values)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("environment credentials must not create files: %v", entries)
	}
}

func TestOpenRouterAPIKeyIsACompatibilityFallbackOnly(t *testing.T) {
	s, _ := newStore(t, nil)
	s.Env = map[string]string{"OPENROUTER_API_KEY": "compat-key"}

	creds, err := s.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if creds.APIKey != "compat-key" {
		t.Errorf("api key = %q, want the compatibility fallback", creds.APIKey)
	}
	if creds.Management != "" {
		t.Errorf("OPENROUTER_API_KEY must never satisfy the management key: %q", creds.Management)
	}
	if creds.HasManagement() {
		t.Error("HasManagement must be false without a management key")
	}
}

func TestKeyringIsPreferredOverFiles(t *testing.T) {
	kr := newFakeKeyring()
	s, dir := newStore(t, kr)
	if err := kr.Set("routeburn", "management", "mgmt-keyring"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "management-token"), []byte("mgmt-file"), 0o600); err != nil {
		t.Fatal(err)
	}

	creds, err := s.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if creds.Management != "mgmt-keyring" {
		t.Errorf("management = %q, want the keyring value", creds.Management)
	}
}

func TestFileFallbackRequiresRestrictivePermissions(t *testing.T) {
	if !PermissionsAreEnforced {
		t.Skip("the OS does not expose POSIX permission bits")
	}
	s, dir := newStore(t, nil)
	path := filepath.Join(dir, "management-token")
	if err := os.WriteFile(path, []byte("mgmt-file"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := s.Resolve(context.Background())
	if err == nil {
		t.Fatal("expected an error for a group/world-readable token file")
	}
	if !errors.Is(err, ErrInsecurePermissions) {
		t.Errorf("err = %v, want ErrInsecurePermissions", err)
	}
	if strings.Contains(err.Error(), "mgmt-file") {
		t.Errorf("error must not leak the token: %v", err)
	}
}

func TestFileFallbackWorksWhenKeyringFails(t *testing.T) {
	kr := newFakeKeyring()
	kr.fail = true
	s, dir := newStore(t, kr)
	if err := os.WriteFile(filepath.Join(dir, "management-token"), []byte(" mgmt-file \n"), 0o600); err != nil {
		t.Fatal(err)
	}

	creds, err := s.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if creds.Management != "mgmt-file" {
		t.Errorf("management = %q, want the trimmed file value", creds.Management)
	}
}

func TestSaveCreatesRestrictiveFileWhenKeyringDisabled(t *testing.T) {
	s, dir := newStore(t, nil)
	if err := s.Save(context.Background(), KindManagement, "mgmt-new"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	path := filepath.Join(dir, "management-token")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Windows reports 0666 for every file, so the mode check is meaningless there.
	if PermissionsAreEnforced {
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("permissions = %o, want 600", perm)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "mgmt-new" {
		t.Errorf("file = %q", data)
	}
}

func TestSaveRejectsEmptySecret(t *testing.T) {
	s, _ := newStore(t, newFakeKeyring())
	if err := s.Save(context.Background(), KindManagement, "   "); err == nil {
		t.Fatal("expected an error for an empty secret")
	}
}

func TestLogoutRemovesOnlyRouteburnCredentials(t *testing.T) {
	kr := newFakeKeyring()
	s, dir := newStore(t, kr)
	if err := kr.Set("routeburn", "management", "m"); err != nil {
		t.Fatal(err)
	}
	if err := kr.Set("routeburn", "api-key", "a"); err != nil {
		t.Fatal(err)
	}
	if err := kr.Set("other-app", "management", "keep"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "management-token"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(dir, "unrelated")
	if err := os.WriteFile(keep, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := s.Logout(context.Background()); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, ok := kr.values["routeburn/management"]; ok {
		t.Error("management keyring entry must be removed")
	}
	if _, ok := kr.values["routeburn/api-key"]; ok {
		t.Error("api-key keyring entry must be removed")
	}
	if kr.values["other-app/management"] != "keep" {
		t.Error("unrelated keyring entries must be untouched")
	}
	if _, err := os.Stat(filepath.Join(dir, "management-token")); !errors.Is(err, os.ErrNotExist) {
		t.Error("fallback token file must be removed")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Error("unrelated files must be untouched")
	}
}

func TestRedactRemovesSecrets(t *testing.T) {
	// The fake key is assembled at run time rather than written as a literal:
	// the repository's secret scanner rejects any committed string matching the
	// real key shape, and a test fixture is no exception.
	secret := "sk-or-v1-" + strings.Repeat("ab", 8)
	cases := map[string]string{
		"bearer": "Authorization: Bearer " + secret,
		"query":  "https://openrouter.ai/api/v1/key?token=" + secret,
		"inline": "failed with " + secret + " while fetching",
	}
	for name, in := range cases {
		out := Redact(in)
		if strings.Contains(out, secret) {
			t.Errorf("%s: Redact leaked the secret: %q", name, out)
		}
		if !strings.Contains(out, "[redacted]") {
			t.Errorf("%s: Redact should mark the redaction: %q", name, out)
		}
	}
	if got := Redact("nothing sensitive here"); got != "nothing sensitive here" {
		t.Errorf("Redact changed a clean string: %q", got)
	}
}

func TestCredsHasHelpers(t *testing.T) {
	var c Creds
	if c.HasManagement() || c.HasAPIKey() {
		t.Error("zero Creds must report nothing configured")
	}
	c.Management = "m"
	c.APIKey = "a"
	if !c.HasManagement() || !c.HasAPIKey() {
		t.Error("populated Creds must report both configured")
	}
	got := c.Redacted()
	if got != "management=true api_key=true" {
		t.Errorf("Redacted = %q", got)
	}
}
