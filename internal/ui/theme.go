package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type CustomTheme struct{}

var _ fyne.Theme = (*CustomTheme)(nil)

// Color returns the color for a named color in the theme.
func (t *CustomTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary:
		return color.NRGBA{
			R: 63,
			G: 81,
			B: 181,
			A: 255,
		}
	case theme.ColorNameSuccess:
		return color.NRGBA{
			R: 76,
			G: 175,
			B: 80,
			A: 255,
		}
	case theme.ColorNameWarning:
		return color.NRGBA{
			R: 255,
			G: 152,
			B: 0,
			A: 255,
		}
	case theme.ColorNameError:
		return color.NRGBA{
			R: 244,
			G: 67,
			B: 54,
			A: 255,
		}
	}

	return theme.DefaultTheme().Color(name, variant)
}

// Icon returns the resource for a named icon in the theme.
func (t *CustomTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

// Font returns the resource for a named font in the theme.
func (t *CustomTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

// Size returns the size for a named size in the theme.
func (t *CustomTheme) Size(name fyne.ThemeSizeName) float32 {

	switch name {
	case theme.SizeNamePadding:
		return 4
	case theme.SizeNameInnerPadding:
		return 4
	}

	return theme.DefaultTheme().Size(name)
}
