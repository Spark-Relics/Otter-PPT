package model

// MediaData holds video or audio media properties.
type MediaData struct {
	MediaPath  string `json:"media_path"`              // local path or URL to media file
	MediaType  string `json:"media_type"`              // "video" or "audio"
	PosterPath string `json:"poster_path,omitempty"`   // cover image for video (local path or URL)
	MimeType   string `json:"mime_type,omitempty"`     // e.g. "video/mp4", "audio/mpeg"
	Loop       bool   `json:"loop,omitempty"`          // loop playback
	AutoPlay   bool   `json:"autoplay,omitempty"`      // start automatically when slide appears
	FileSize   int64  `json:"file_size,omitempty"`     // bytes (auto-detected if 0)
	Duration   float64 `json:"duration,omitempty"`     // seconds (informational)
	Title      string `json:"title,omitempty"`         // media title shown in player
}
