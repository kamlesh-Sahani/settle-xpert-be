package assets

import "embed"

// Files embeds the logo and PDF fonts directly into the binary so PDF generation
// works in production even when the working directory has no local assets folder.
//
//go:embed settlexpert-logo.png fonts/*.ttf
var Files embed.FS
