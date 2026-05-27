package instances_test

import (
	"os"
	"testing"

	"github.com/t0mer/AGHexporter/internal/instances"
)

// --- helpers ---

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "secret-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(f.Name()) })
	_, _ = f.WriteString(content)
	f.Close()
	return f.Name()
}

// --- ResolvePort tests ---

func TestPortPrecedence_EnvWins(t *testing.T) {
	t.Setenv("ADGUARD_EXPORTER_PORT", "9999")
	got := instances.ResolvePort(9100)
	if got != 9999 {
		t.Errorf("want 9999, got %d", got)
	}
}

func TestPortPrecedence_FlagUsed(t *testing.T) {
	t.Setenv("ADGUARD_EXPORTER_PORT", "")
	got := instances.ResolvePort(8080)
	if got != 8080 {
		t.Errorf("want 8080, got %d", got)
	}
}

// --- Format A tests ---

func TestFormatA_Inline(t *testing.T) {
	t.Setenv("ADGUARD_URL_1", "http://192.168.1.1")
	t.Setenv("ADGUARD_USERNAME_1", "admin")
	t.Setenv("ADGUARD_PASSWORD_1", "secret")

	insts, err := instances.DiscoverInstances(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(insts) != 1 {
		t.Fatalf("want 1 instance, got %d", len(insts))
	}
	if insts[0].Name != "192.168.1.1" {
		t.Errorf("want name 192.168.1.1, got %s", insts[0].Name)
	}
	if insts[0].Username != "admin" {
		t.Errorf("want username admin, got %s", insts[0].Username)
	}
}

func TestFormatA_MultiIndex(t *testing.T) {
	t.Setenv("ADGUARD_URL_1", "http://host1")
	t.Setenv("ADGUARD_USERNAME_1", "admin")
	t.Setenv("ADGUARD_PASSWORD_1", "pass1")
	t.Setenv("ADGUARD_NAME_1", "primary")

	t.Setenv("ADGUARD_URL_2", "http://host2")
	t.Setenv("ADGUARD_USERNAME_2", "admin")
	t.Setenv("ADGUARD_PASSWORD_2", "pass2")
	t.Setenv("ADGUARD_NAME_2", "secondary")

	insts, err := instances.DiscoverInstances(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(insts) != 2 {
		t.Fatalf("want 2 instances, got %d", len(insts))
	}
	if insts[0].Name != "primary" || insts[1].Name != "secondary" {
		t.Errorf("wrong names: %s, %s", insts[0].Name, insts[1].Name)
	}
}

func TestFormatA_SecretFile(t *testing.T) {
	userFile := writeTempFile(t, "admin\n")
	passFile := writeTempFile(t, "secret\n")

	t.Setenv("ADGUARD_URL_1", "http://192.168.1.1")
	t.Setenv("ADGUARD_USERNAME_FILE_1", userFile)
	t.Setenv("ADGUARD_PASSWORD_FILE_1", passFile)

	insts, err := instances.DiscoverInstances(nil)
	if err != nil {
		t.Fatal(err)
	}
	if insts[0].Username != "admin" {
		t.Errorf("want admin, got %q", insts[0].Username)
	}
	if insts[0].Password != "secret" {
		t.Errorf("want secret, got %q", insts[0].Password)
	}
}

func TestFormatA_BothPasswordFormsError(t *testing.T) {
	passFile := writeTempFile(t, "secret\n")

	t.Setenv("ADGUARD_URL_1", "http://192.168.1.1")
	t.Setenv("ADGUARD_USERNAME_1", "admin")
	t.Setenv("ADGUARD_PASSWORD_1", "secret")
	t.Setenv("ADGUARD_PASSWORD_FILE_1", passFile)

	_, err := instances.DiscoverInstances(nil)
	if err == nil {
		t.Fatal("want error when both PASSWORD and PASSWORD_FILE are set, got nil")
	}
}

func TestFormatA_BothUsernameFormsError(t *testing.T) {
	userFile := writeTempFile(t, "admin\n")

	t.Setenv("ADGUARD_URL_1", "http://192.168.1.1")
	t.Setenv("ADGUARD_USERNAME_1", "admin")
	t.Setenv("ADGUARD_USERNAME_FILE_1", userFile)
	t.Setenv("ADGUARD_PASSWORD_1", "secret")

	_, err := instances.DiscoverInstances(nil)
	if err == nil {
		t.Fatal("want error when both USERNAME and USERNAME_FILE are set, got nil")
	}
}

// --- Format B tests ---

func TestFormatB_Happy(t *testing.T) {
	t.Setenv("ADGUARD_URLS", "http://host1,http://host2")
	t.Setenv("ADGUARD_USERNAMES", "admin,admin")
	t.Setenv("ADGUARD_PASSWORDS", "pass1,pass2")

	insts, err := instances.DiscoverInstances(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(insts) != 2 {
		t.Fatalf("want 2, got %d", len(insts))
	}
}

func TestFormatB_WithNames(t *testing.T) {
	t.Setenv("ADGUARD_URLS", "http://host1,http://host2")
	t.Setenv("ADGUARD_USERNAMES", "admin,admin")
	t.Setenv("ADGUARD_PASSWORDS", "pass1,pass2")
	t.Setenv("ADGUARD_NAMES", "alpha,beta")

	insts, err := instances.DiscoverInstances(nil)
	if err != nil {
		t.Fatal(err)
	}
	if insts[0].Name != "alpha" || insts[1].Name != "beta" {
		t.Errorf("want alpha,beta got %s,%s", insts[0].Name, insts[1].Name)
	}
}

func TestFormatB_LengthMismatch(t *testing.T) {
	t.Setenv("ADGUARD_URLS", "http://host1,http://host2")
	t.Setenv("ADGUARD_USERNAMES", "admin")
	t.Setenv("ADGUARD_PASSWORDS", "pass1,pass2")

	_, err := instances.DiscoverInstances(nil)
	if err == nil {
		t.Fatal("want error for length mismatch, got nil")
	}
}

// --- Format C tests ---

func TestFormatC_Happy(t *testing.T) {
	insts, err := instances.DiscoverInstances([]string{
		"url=http://host1,username=admin,password=pass1",
		"url=http://host2,username=admin,password=pass2,name=host2-alias",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(insts) != 2 {
		t.Fatalf("want 2, got %d", len(insts))
	}
	if insts[1].Name != "host2-alias" {
		t.Errorf("want host2-alias, got %s", insts[1].Name)
	}
}

func TestFormatC_MissingURL(t *testing.T) {
	_, err := instances.DiscoverInstances([]string{"username=admin,password=pass"})
	if err == nil {
		t.Fatal("want error for missing url, got nil")
	}
}

func TestFormatC_BothCredentialFormsError(t *testing.T) {
	passFile := writeTempFile(t, "pass\n")
	_, err := instances.DiscoverInstances([]string{
		"url=http://host,username=admin,password=pass,password_file=" + passFile,
	})
	if err == nil {
		t.Fatal("want error for both password and password_file, got nil")
	}
}

// --- Combined sources test ---

func TestCombined_AllFormats(t *testing.T) {
	t.Setenv("ADGUARD_URL_1", "http://host-a")
	t.Setenv("ADGUARD_USERNAME_1", "admin")
	t.Setenv("ADGUARD_PASSWORD_1", "pass")
	t.Setenv("ADGUARD_NAME_1", "format-a")

	t.Setenv("ADGUARD_URLS", "http://host-b")
	t.Setenv("ADGUARD_USERNAMES", "admin")
	t.Setenv("ADGUARD_PASSWORDS", "pass")
	t.Setenv("ADGUARD_NAMES", "format-b")

	flags := []string{"url=http://host-c,username=admin,password=pass,name=format-c"}

	insts, err := instances.DiscoverInstances(flags)
	if err != nil {
		t.Fatal(err)
	}
	if len(insts) != 3 {
		t.Fatalf("want 3, got %d", len(insts))
	}
	names := []string{insts[0].Name, insts[1].Name, insts[2].Name}
	want := []string{"format-a", "format-b", "format-c"}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("position %d: want %s, got %s", i, want[i], names[i])
		}
	}
}

func TestDuplicateNames_Error(t *testing.T) {
	t.Setenv("ADGUARD_URL_1", "http://host1")
	t.Setenv("ADGUARD_USERNAME_1", "admin")
	t.Setenv("ADGUARD_PASSWORD_1", "pass")
	t.Setenv("ADGUARD_NAME_1", "same-name")

	t.Setenv("ADGUARD_URLS", "http://host2")
	t.Setenv("ADGUARD_USERNAMES", "admin")
	t.Setenv("ADGUARD_PASSWORDS", "pass")
	t.Setenv("ADGUARD_NAMES", "same-name")

	_, err := instances.DiscoverInstances(nil)
	if err == nil {
		t.Fatal("want duplicate name error, got nil")
	}
}

func TestZeroInstances_Error(t *testing.T) {
	_, err := instances.DiscoverInstances(nil)
	if err == nil {
		t.Fatal("want error for zero instances, got nil")
	}
}
