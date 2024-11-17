package config

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	Colors struct {
		Text      string `json:"text"`
		Border    string `json:"border"`
		Highlight string `json:"highlight"`
		Command   string `json:"command"`
		Blinking  string `json:"blinking"`
		Label     string `json:"label"`
		Value     string `json:"value"`
		Focused   string `json:"focused"`
	} `json:"colors"`
	Navigation struct {
		Up    string `json:"up"`
		Down  string `json:"down"`
		Left  string `json:"left"`
		Right string `json:"right"`
	} `json:"navigation"`
	KeyBindings struct {
		Quit        string `json:"quit"`
		NextBox     string `json:"nextBox"`
		PreviousBox string `json:"previousBox"`
		SelectUp    string `json:"selectUp"`
		SelectDown  string `json:"selectDown"`
		Execute     string `json:"execute"`
		Backspace   string `json:"backspace"`
		Rename      string `json:"rename"`
		Move        string `json:"move"`
		Delete      string `json:"delete"`
		Copy        string `json:"copy"`
	} `json:"keyBindings"`
	Theme  string `json:"theme"`
	Themes struct {
		Dark struct {
			Background string `json:"background"`
			Foreground string `json:"foreground"`
			Highlight  string `json:"highlight"`
			Cursor     string `json:"cursor"`
		} `json:"dark"`
		Light struct {
			Background string `json:"background"`
			Foreground string `json:"foreground"`
			Highlight  string `json:"highlight"`
			Cursor     string `json:"cursor"`
		} `json:"light"`
	} `json:"themes"`
	Font struct {
		Size  int    `json:"size"`
		Style string `json:"style"`
	} `json:"font"`
	FileFilters struct {
		ShowHiddenFiles bool     `json:"showHiddenFiles"`
		FileExtensions  []string `json:"fileExtensions"`
	} `json:"fileFilters"`
	Notifications struct {
		Enabled  bool `json:"enabled"`
		Duration int  `json:"duration"`
	} `json:"notifications"`
	Language string `json:"language"`
	AutoSave struct {
		Enabled  bool `json:"enabled"`
		Interval int  `json:"interval"`
	} `json:"autoSave"`
	Logging struct {
		Level string `json:"level"`
		File  string `json:"file"`
	} `json:"logging"`
	PreferredEditor string `json:"preferredEditor"`
}

type FileInfo struct {
	Name           string
	Permissions    string
	Owner          string
	IsExecutable   bool
	IsSymlink      bool
	SymlinkTarget  string
	MountPoint     string
	SELinuxContext string
	GitRepoStatus  string
	LastAccessTime string
	CreationTime   string
	Size           int64
	FileType       string
	Inode          uint64
	HardLinksCount uint64
}

func LoadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}