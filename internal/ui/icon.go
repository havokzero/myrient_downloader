package ui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed icon.png
var iconBytes []byte

// AppIcon is the embedded application icon resource.
var AppIcon = fyne.NewStaticResource("app-icon", iconBytes)
