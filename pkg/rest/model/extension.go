package model

type InstallExtension struct {
	URL string `json:"url"`
}

type UpdateExtensionSettings struct {
	Identity string         `json:"identity"`
	Settings map[string]any `json:"settings"`
}

type UpgradeCheckExtensionResp struct {
	NewVersion string `json:"newVersion"`
}
