package api

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/GopeedLab/gopeed/pkg/download"
	restmodel "github.com/GopeedLab/gopeed/pkg/rest/model"
)

func TestExtensionPasswordIsMaskedInAPI(t *testing.T) {
	d := download.NewDownloader(&download.DownloaderConfig{StorageDir: t.TempDir(), TempDir: t.TempDir()})
	if err := d.Setup(); err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	ext, err := d.InstallExtensionByFolder("../download/testdata/extensions/settings_empty", true)
	if err != nil {
		t.Fatal(err)
	}
	ext.Settings = append(ext.Settings,
		&download.Setting{Name: "token", Type: download.SettingTypePassword},
		&download.Setting{Name: "display", Type: download.SettingTypeString, Value: "public"},
	)
	if err := d.UpdateExtensionSettings(ext.Identity, map[string]any{"token": "private-token"}); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(d)
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"/api/v1/extensions", "/api/v1/extensions/" + ext.Identity} {
		status, body := service.JSON(http.MethodGet, path, nil, nil, nil)
		if status != http.StatusOK || !bytes.Contains(body, []byte(download.PasswordMask)) {
			t.Fatalf("%s: expected a masked password, got %d %s", path, status, body)
		}
		if bytes.Contains(body, []byte("private-token")) {
			t.Fatalf("%s leaked the secret in its JSON: %s", path, body)
		}
	}
	serviceTestRequest[any](t, service, http.MethodPut,
		"/api/v1/extensions/"+ext.Identity+"/settings",
		&restmodel.UpdateExtensionSettings{Settings: map[string]any{"token": download.PasswordMask}},
	)
	if got := ext.Settings[0].Value; got != "private-token" {
		t.Fatalf("API round-trip overwrote password: %q", got)
	}
	serviceTestRequest[any](t, service, http.MethodPut,
		"/api/v1/extensions/"+ext.Identity+"/settings",
		&restmodel.UpdateExtensionSettings{Settings: map[string]any{"token": "changed-token"}},
	)
	if got := ext.Settings[0].Value; got != "changed-token" {
		t.Fatalf("API failed to update password: %q", got)
	}
}
