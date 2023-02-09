package extension

type Extension struct {
	// URL git repository url
	URL string `json:"url"`
	// Dir extension directory name
	Dir string `json:"dir"`

	// Manifest extension manifest info
	*Manifest `json:"manifest"`
}

type Manifest struct {
	// Name extension global unique name
	Name string `json:"name"`
	// title
	Title string `json:"title"`
	// Icon
	Icon string `json:"icon"`
	// Version
	Version float32 `json:"version"`
	// About
	About string `json:"about"`
	// Homepage
	Homepage string `json:"homepage"`
	// Hooks
	Hooks []*Hook `json:"hooks"`
	// Settings
	Settings []*Setting `json:"settings"`
}

type Hook struct {
	// BeforeResolve before resolve hook
	BeforeResolve *HookFunc `json:"beforeResolve"`
}

type HookFunc struct {
	// Regexps match url
	Regexps string `json:"regexps"`
	// Script script file path
	Script string `json:"script"`
}

type Setting struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Required bool   `json:"required"`
	// setting type, support: String, number, Boolean
	Type string `json:"type"`
	// default value
	value   any       `json:"value"`
	Options []*Option `json:"options"`
}

type Option struct {
	Title string `json:"title"`
	Value any    `json:"value"`
}
