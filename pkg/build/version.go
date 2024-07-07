package build

import "fmt"

var (
	AppName          = "cloudzero-agent-validator"
	AuthorName       = "Cloudzero"
	ChartsRepo       = "cloudzero-charts"
	AuthorEmail      = "support@cloudzero.com"
	Copyright        = "© 2024 Cloudzero, Inc."
	PlatformEndpoint = "https://api.cloudzero.com"
)

func GetVersion() string {
	return fmt.Sprintf("%s.%s.%s-%s", AppName, Rev, Tag, Time)
}
