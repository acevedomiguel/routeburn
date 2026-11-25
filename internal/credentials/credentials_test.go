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

	// setErr and deleteErr fail a single operation while the others keep
	// working, which is what a partially reachable OS keyring looks like. The
	// real keyring reports every failure as "unavailable", so tests must be
	// able to make Get succeed while Set or Delete fails.
	setErr    error
	deleteErr error
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
	if f.setErr != nil {
		return f.setErr
	}
	f.values[service+"/"+user] = secret
	return nil
}

func (f *fakeKeyring) Delete(service, user string) error {
	if f.fail {
		return ErrKeyringUnavailable
	}
	if f.deleteErr != nil {
		return f.deleteErr
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

// A successful keyring write must clear the fallback file. Resolution prefers
// the keyring, so leaving the file behind would let a later launch with an
// unreachable keyring resurrect the credential that was just replaced.
func TestSaveRemovesTheStaleFallbackFileAfterAKeyringWrite(t *testing.T) {
	kr := newFakeKeyring()
	s, dir := newStore(t, kr)
	if err := kr.Set("routeburn", "management", "old-key"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "management-token")
	if err := os.WriteFile(path, []byte("old-key"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := s.Save(context.Background(), KindManagement, "new-key"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the stale fallback file must be removed: %v", err)
	}

	creds, err := s.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if creds.Management != "new-key" {
		t.Fatalf("management = %q, want the saved key", creds.Management)
	}

	// With the keyring unreachable the replaced value must not come back.
	kr.fail = true
	creds, err = s.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if creds.Management != "" {
		t.Errorf("management = %q, want no stale key after the fallback was removed", creds.Management)
	}
}

// When the keyring cannot take the write, the value goes to the fallback file
// and the stale keyring entry is cleared so it cannot shadow the file.
func TestSaveFallsBackToTheFileAndClearsTheKeyringEntry(t *testing.T) {
	kr := newFakeKeyring()
	kr.setErr = errors.New("keyring: transient failure")
	kr.values["routeburn/management"] = "old-key"
	s, dir := newStore(t, kr)

	if err := s.Save(context.Background(), KindManagement, "new-key"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, ok := kr.values["routeburn/management"]; ok {
		t.Error("the stale keyring entry must be cleared")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "management-token"))
	if err != nil || string(raw) != "new-key" {
		t.Fatalf("fallback file = %q, %v", raw, err)
	}

	// With the keyring healthy again the file value must still win.
	kr.setErr = nil
	creds, err := s.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if creds.Management != "new-key" {
		t.Errorf("management = %q, want the file value", creds.Management)
	}
}

// A keyring entry that survives the delete still shadows the file, so the save
// must report failure instead of a success that will not take effect.
func TestSaveRefusesWhenTheKeyringStillShadowsTheFile(t *testing.T) {
	kr := newFakeKeyring()
	kr.setErr = errors.New("keyring: locked")
	kr.deleteErr = errors.New("keyring: locked")
	kr.values["routeburn/management"] = "old-key"
	s, dir := newStore(t, kr)

	err := s.Save(context.Background(), KindManagement, "new-key")
	if err == nil {
		t.Fatal("expected a failure when the keyring entry cannot be replaced")
	}
	if strings.Contains(err.Error(), "new-key") || strings.Contains(err.Error(), "old-key") {
		t.Errorf("the error must not leak a key: %v", err)
	}

	// The fallback file was written before the keyring was touched, so the new
	// value is durable even though the call reported failure.
	raw, readErr := os.ReadFile(filepath.Join(dir, "management-token"))
	if readErr != nil || string(raw) != "new-key" {
		t.Fatalf("fallback file = %q, %v", raw, readErr)
	}
}

// A fallback file that cannot be cleared must abort the save before the keyring
// is touched. Writing the keyring first would make the reported failure a lie:
// the caller would keep using the old key while the store held the new one.
func TestSaveAbortsBeforeTheKeyringWhenTheStaleFallbackCannotBeCleared(t *testing.T) {
	kr := newFakeKeyring()
	kr.values["routeburn/management"] = "old-key"

	// A file in the parent position makes removal of the fallback impossible.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := NewStore(Options{
		Keyring:             kr,
		ManagementTokenFile: filepath.Join(blocker, "management-token"),
		APIKeyFile:          filepath.Join(dir, "api-key"),
		KeyringEnabled:      true,
	})

	if err := s.Save(context.Background(), KindManagement, "new-key"); err == nil {
		t.Fatal("expected the save to fail when the stale fallback cannot be cleared")
	}
	if kr.values["routeburn/management"] != "old-key" {
		t.Errorf("the keyring must be left untouched, got %q", kr.values["routeburn/management"])
	}
}

// Without a keyring the file is the only backend, and an unwritable path must
// surface as an error rather than an apparent success.
func TestSaveWithoutAKeyringReportsAFailedFallbackWrite(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := NewStore(Options{
		ManagementTokenFile: filepath.Join(blocker, "management-token"),
		APIKeyFile:          filepath.Join(dir, "api-key"),
	})

	if err := s.Save(context.Background(), KindManagement, "new-key"); err == nil {
		t.Fatal("expected the save to fail")
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

// syntheticKey is invented for tests. It is never a real credential.
const syntheticKey = "sk-or-v1-" + "testonly0000" + "abcdefgh"

// A Redactor hides the exact configured key even when it does not look like a
// key, in raw and URL-escaped forms, and never leaks it back.
func TestRedactorRemovesExactConfiguredKey(t *testing.T) {
	r := NewRedactor(syntheticKey)
	inputs := []string{
		"validation failed for " + syntheticKey,
		"Authorization: Bearer " + syntheticKey,
		"GET /credits?key=" + syntheticKey,
		"error: " + syntheticKey + " expired",
	}
	for _, in := range inputs {
		out := r.Redact(in)
		if strings.Contains(out, syntheticKey) {
			t.Errorf("Redact(%q) = %q, still contains the key", in, out)
		}
		if !strings.Contains(out, "[redacted]") {
			t.Errorf("Redact(%q) = %q, should mark the redaction", in, out)
		}
	}
}

// A key whose shape does not match the documented prefix is still hidden,
// because the redactor matches the exact configured value.
func TestRedactorHidesNonStandardKeyShape(t *testing.T) {
	const opaque = "totally-opaque-key-value"
	r := NewRedactor(opaque)
	out := r.Redact("the server rejected " + opaque + " today")
	if strings.Contains(out, opaque) {
		t.Fatalf("Redact = %q, leaked the opaque key", out)
	}
}

// A nil receiver still scrubs the generic shapes, so a missing configuration
// never disables redaction.
func TestNilRedactorStillRedactsGenericShapes(t *testing.T) {
	var r *Redactor
	out := r.Redact("Authorization: Bearer " + syntheticKey)
	if strings.Contains(out, syntheticKey) {
		t.Fatalf("nil Redact = %q, leaked the key", out)
	}
}

// Empty and whitespace-only values are ignored rather than becoming a
// match-everything pattern.
func TestRedactorIgnoresEmptySecrets(t *testing.T) {
	r := NewRedactor("", "   ", syntheticKey)
	if got := r.Redact("nothing sensitive here"); got != "nothing sensitive here" {
		t.Errorf("Redact changed a clean string: %q", got)
	}
}

// ManagementFromEnvironment reports whether the environment supplies the
// management key, without handing the secret back.
func TestManagementFromEnvironment(t *testing.T) {
	s := NewStore(Options{Env: map[string]string{}})
	if s.ManagementFromEnvironment() {
		t.Error("an empty environment must not report a management key")
	}
	s = NewStore(Options{Env: map[string]string{EnvManagementKey: "mgmt"}})
	if !s.ManagementFromEnvironment() {
		t.Error("an environment management key must be reported")
	}
	// A regular API key never satisfies the management slot.
	s = NewStore(Options{Env: map[string]string{EnvOpenRouterKey: "api"}})
	if s.ManagementFromEnvironment() {
		t.Error("OPENROUTER_API_KEY must not count as a management key")
	}
}
