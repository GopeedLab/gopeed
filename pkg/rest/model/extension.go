package model

type InstallExtension struct {
	URL string `json:"url"`
}

type UpdateExtensionSettings struct {
	Settings map[string]any `json:"settings"`
}

type SwitchExtension struct {
	Status bool `json:"status"`
}

type UpdateCheckExtensionResp struct {
	NewVersion string `json:"newVersion"`
}
