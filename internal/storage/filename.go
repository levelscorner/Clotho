package storage

import (
	"strings"
)

// mimeToExt maps common content types to their preferred filename extension.
// Keep this list intentionally small — providers produce a known set of media
// types. "Unknown" falls through to "bin".
var mimeToExt = map[string]string{
	"image/png":                "png",
	"image/jpeg":               "jpg",
	"image/jpg":                "jpg",
	"image/webp":               "webp",
	"image/gif":                "gif",
	"audio/mpeg":               "mp3",
	"audio/mp3":                "mp3",
	"audio/wav":                "wav",
	"audio/x-wav":              "wav",
	"audio/ogg":                "ogg",
	"audio/webm":               "webm",
	"video/mp4":                "mp4",
	"video/webm":               "webm",
	"video/quicktime":          "mov",
	"application/json":         "json",
	"application/octet-stream": "bin",
}

// extToMIME is the inverse for serving files back with a sensible
// Content-Type header.
var extToMIME = map[string]string{
	"png":  "image/png",
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
	"webp": "image/webp",
	"gif":  "image/gif",
	"mp3":  "audio/mpeg",
	"wav":  "audio/wav",
	"ogg":  "audio/ogg",
	"mp4":  "video/mp4",
	"mov":  "video/quicktime",
	"webm": "video/webm",
	"json": "application/json",
	"txt":  "text/plain; charset=utf-8",
}

// ExtensionForMIME returns a filename extension for the given content type.
// Unknown types fall back to "bin". The returned value never includes a
// leading dot.
func ExtensionForMIME(contentType string) string {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	// Strip parameters like "; charset=utf-8".
	if idx := strings.Index(ct, ";"); idx >= 0 {
		ct = strings.TrimSpace(ct[:idx])
	}
	if ext, ok := mimeToExt[ct]; ok {
		return ext
	}
	return "bin"
}

// ContentTypeForExtension returns a best-effort MIME type for the given file
// extension (with or without leading dot). Unknown extensions fall back to
// "application/octet-stream".
func ContentTypeForExtension(ext string) string {
	e := strings.ToLower(strings.TrimSpace(ext))
	e = strings.TrimPrefix(e, ".")
	if mime, ok := extToMIME[e]; ok {
		return mime
	}
	return "application/octet-stream"
}

// Slugify returns a minimal filesystem-safe version of s. Characters outside
// [a-zA-Z0-9._-] are replaced with '-'. Consecutive dashes are collapsed and
// leading/trailing dashes trimmed. An empty or fully stripped input returns
// "" so callers can apply a fallback.
func Slugify(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	prevDash := false
	for _, r := range s {
		if isSafeSlugRune(r) {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func isSafeSlugRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9':
		return true
	case r == '.' || r == '_' || r == '-':
		return true
	}
	return false
}

// isSafeFilename reports whether name is acceptable as a leaf filename.
// Rejects empty strings, dotfiles (leading "."), path separators, and any
// ".." segment.
func isSafeFilename(name string) bool {
	if name == "" {
		return false
	}
	if strings.HasPrefix(name, ".") {
		return false
	}
	if strings.ContainsAny(name, `/\`) {
		return false
	}
	if name == ".." || strings.Contains(name, "..") {
		return false
	}
	return true
}
