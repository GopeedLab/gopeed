package download

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExtensionPublicViewMasksPasswords(t *testing.T) {
	ext := &Extension{Settings: []*Setting{
		{Name: "token", Type: SettingTypePassword, Value: "real-secret"},
		{Name: "empty", Type: SettingTypePassword, Value: ""},
		{Name: "normal", Type: SettingTypeString, Value: "visible"},
	}}
	view := ext.PublicView()
	if got := view.Settings[0].Value; got != PasswordMask {
		t.Fatalf("masked value = %q, want %q", got, PasswordMask)
	}
	if got := view.Settings[1].Value; got != "" {
		t.Fatalf("empty password = %q, want empty", got)
	}
	if got := view.Settings[2].Value; got != "visible" {
		t.Fatalf("ordinary value = %q, want visible", got)
	}
	if ext.Settings[0].Value != "real-secret" {
		t.Fatal("masking changed the installed extension's password")
	}
	view.Settings[0].Value = "modified"
	if ext.Settings[0].Value != "real-secret" {
		t.Fatal("changing the public copy modified the installed extension")
	}
	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "real-secret") {
		t.Fatalf("API response leaked a password: %s", encoded)
	}
	if got := parseSettings(ext.Settings)["token"]; got != "real-secret" {
		t.Fatalf("extension runtime got %q, want the real password", got)
	}
}

func TestExtensionPasswordUpdatesPreserveMaskedValues(t *testing.T) {
	setupDownloader(func(d *Downloader) {
		ext, err := d.InstallExtensionByFolder("./testdata/extensions/settings_empty", false)
		if err != nil {
			t.Fatal(err)
		}
		ext.Settings = append(ext.Settings, &Setting{Name: "token", Type: SettingTypePassword})
		if err := d.UpdateExtensionSettings(ext.Identity, map[string]any{"token": "first-secret"}); err != nil {
			t.Fatal(err)
		}
		if err := d.UpdateExtensionSettings(ext.Identity, map[string]any{"token": PasswordMask}); err != nil {
			t.Fatal(err)
		}
		if got := ext.Settings[0].Value; got != "first-secret" {
			t.Fatalf("masked update lost password: %q", got)
		}
		if err := d.UpdateExtensionSettings(ext.Identity, map[string]any{"token": "second-secret"}); err != nil {
			t.Fatal(err)
		}
		if got := ext.Settings[0].Value; got != "second-secret" {
			t.Fatalf("password edit was not saved: %q", got)
		}
		if err := d.UpdateExtensionSettings(ext.Identity, map[string]any{"token": ""}); err != nil {
			t.Fatal(err)
		}
		if got := ext.Settings[0].Value; got != "" {
			t.Fatalf("clearing password failed: %q", got)
		}
	})
}
