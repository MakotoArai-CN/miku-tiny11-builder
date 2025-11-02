package types

type BuildMode string

const (
	ModeStandard BuildMode = "standard"
	ModeCore     BuildMode = "core"
	ModeNano     BuildMode = "nano"
)

type BuildRequest struct {
	ISODrive       string      `json:"isoDrive"`
	ScratchDrive   string      `json:"scratchDrive,omitempty"`
	Mode           BuildMode   `json:"mode"`
	Theme          string      `json:"theme"`
	ImageIndex     int         `json:"imageIndex,omitempty"`
	OutputISO      string      `json:"outputIso,omitempty"`
	PreinstallApps []string    `json:"preinstallApps,omitempty"`
	UseESD         bool        `json:"useEsd,omitempty"`
	Verbose        bool        `json:"verbose,omitempty"`
}

type BuildStatus struct {
	Phase      string  `json:"phase"`
	Progress   float64 `json:"progress"`
	Message    string  `json:"message"`
	IsComplete bool    `json:"isComplete"`
	Error      string  `json:"error,omitempty"`
	OutputISO  string  `json:"outputIso,omitempty"`
}

type BuildResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	OutputISO string `json:"outputIso,omitempty"`
	Error     string `json:"error,omitempty"`
}