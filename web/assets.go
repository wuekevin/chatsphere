package web

import "embed"

//go:embed installer
var InstallerFS embed.FS

//go:embed widget
var WidgetFS embed.FS
