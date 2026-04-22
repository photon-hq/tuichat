// Package kitty detects terminal support for the Kitty graphics protocol
// and emits its APC escape sequences for unicode-placeholder image preview.
package kitty

import "os"

// Supported reports whether the current terminal supports Kitty graphics
// with the unicode-placeholder (`U=1`) mode we use.
func Supported() bool {
	if os.Getenv("TUICHAT_DISABLE_IMAGES") == "1" {
		return false
	}
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	if os.Getenv("TERM") == "xterm-kitty" {
		return true
	}
	switch os.Getenv("TERM_PROGRAM") {
	case "ghostty", "WezTerm":
		return true
	}
	if os.Getenv("GHOSTTY_RESOURCES_DIR") != "" {
		return true
	}
	return false
}

// SupportedMimeType reports whether an image MIME is renderable via Kitty.
func SupportedMimeType(mime string) bool {
	switch mime {
	case "image/png", "image/jpeg", "image/jpg", "image/gif", "image/webp", "image/bmp":
		return true
	}
	return false
}
