package schema

import "github.com/ppacher/system-conf/conf"

// Config describes the configuration structure
// parsed by ConfigSpec.
type Config struct {
	DatabasePath string
}

// ConfigSpec describes all valid configuration stanzas
// of the global configuration section.
var ConfigSpec = conf.SectionSpec{
	{
		Name:        "DatabasePath",
		Description: "Path to the ConsoleDB database",
		Type:        conf.StringType,
		Required:    true,
	},
	{
		Name:        "AccessLogPath",
		Description: "Path to the access log file",
		Type:        conf.StringType,
	},
}
