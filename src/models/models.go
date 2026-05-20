package models

type Config map[string]struct {
	Path string   `yaml:"Path"`
	Info []string `yaml:"INFO"`
}

type ProcessRequest struct {
	Filename      string `json:"filename"`
	Year          string `json:"year"`
	Month         string `json:"month"`
	Day           string `json:"day"`
	Correspondent string `json:"correspondent"`
	Info          string `json:"info"`
	Extra         string `json:"extra"`
	Compress      bool   `json:"compress"`
}

type NewEntryRequest struct {
	Correspondent string `json:"correspondent"`
	Info          string `json:"info"`
}

type NextResponse struct {
	Filename      string              `json:"filename"`
	Year          string              `json:"year"`
	Month         string              `json:"month"`
	Day           string              `json:"day"`
	SuggestedCorr string              `json:"suggested_correspondent"`
	SuggestedInfo string              `json:"suggested_info"`
	ConfigData    map[string][]string `json:"config_data"`
	FileSize      string              `json:"file_size"`
	PPI           string              `json:"ppi"`
}
