package fonts

// BUILT_IN_FONTS lists recommended font configurations that don't require
// embedding because they are typically available on the target system.
// Used as fallback when no custom font files are installed.

type RecFont struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Category    string `json:"category"` // sans-serif, serif, mono, cjk, display
	Tags        string `json:"tags"`
	CJK         bool   `json:"cjk"`
	Available   string `json:"availability"` // system, google-fonts, bundled
}

var BuiltInFonts = []RecFont{
	// ── System fonts (widely available) ──
	{"Segoe UI", "Segoe UI", "sans-serif", "modern, clean, ui", false, "system"},
	{"Arial", "Arial", "sans-serif", "universal, clean", false, "system"},
	{"Calibri", "Calibri", "sans-serif", "default office, clean", false, "system"},
	{"Georgia", "Georgia", "serif", "elegant, readable", false, "system"},
	{"Times New Roman", "Times New Roman", "serif", "classic, formal", false, "system"},
	{"Helvetica", "Helvetica", "sans-serif", "classic, clean", false, "system"},
	{"Courier New", "Courier New", "mono", "code, technical", false, "system"},

	// ── Windows CJK ──
	{"Microsoft YaHei", "微软雅黑", "cjk", "modern, ui, chinese", true, "system"},
	{"SimHei", "黑体", "cjk", "bold, poster, chinese", true, "system"},
	{"SimSun", "宋体", "cjk", "classic, body, chinese", true, "system"},
	{"KaiTi", "楷体", "cjk", "elegant, calligraphy, chinese", true, "system"},

	// ── macOS CJK ──
	{"PingFang SC", "苹方", "cjk", "modern, clean, chinese", true, "system"},

	// ── Recommended Google Fonts (downloadable) ──
	{"Inter", "Inter", "sans-serif", "modern, ui, tech", false, "google-fonts"},
	{"Roboto", "Roboto", "sans-serif", "clean, material, ui", false, "google-fonts"},
	{"Open Sans", "Open Sans", "sans-serif", "friendly, readable", false, "google-fonts"},
	{"Montserrat", "Montserrat", "sans-serif", "geometric, bold, display", false, "google-fonts"},
	{"Lato", "Lato", "sans-serif", "warm, professional", false, "google-fonts"},
	{"Poppins", "Poppins", "sans-serif", "geometric, modern, playful", false, "google-fonts"},
	{"Raleway", "Raleway", "sans-serif", "elegant, thin, display", false, "google-fonts"},
	{"Playfair Display", "Playfair Display", "serif", "elegant, editorial, luxury", false, "google-fonts"},
	{"Merriweather", "Merriweather", "serif", "readable, body, editorial", false, "google-fonts"},
	{"Lora", "Lora", "serif", "elegant, editorial", false, "google-fonts"},
	{"Cormorant Garamond", "Cormorant Garamond", "serif", "luxury, refined, display", false, "google-fonts"},
	{"Source Code Pro", "Source Code Pro", "mono", "code, technical", false, "google-fonts"},
	{"JetBrains Mono", "JetBrains Mono", "mono", "code, modern", false, "google-fonts"},
	{"Fira Code", "Fira Code", "mono", "code, ligatures", false, "google-fonts"},
	{"Pacifico", "Pacifico", "script", "handwriting, playful, display", false, "google-fonts"},
	{"Dancing Script", "Dancing Script", "script", "handwriting, elegant", false, "google-fonts"},
	{"Caveat", "Caveat", "script", "handwriting, casual", false, "google-fonts"},
	{"Bebas Neue", "Bebas Neue", "display", "condensed, bold, poster", false, "google-fonts"},
	{"Anton", "Anton", "display", "bold, impact, headline", false, "google-fonts"},
	{"Oswald", "Oswald", "display", "condensed, headline", false, "google-fonts"},

	// ── CJK Google Fonts ──
	{"Noto Sans SC", "思源黑体", "cjk", "modern, clean, chinese", true, "google-fonts"},
	{"Noto Serif SC", "思源宋体", "cjk", "elegant, serif, chinese", true, "google-fonts"},
	{"Noto Sans JP", "Noto Sans JP", "cjk", "modern, japanese", true, "google-fonts"},
	{"Noto Sans KR", "Noto Sans KR", "cjk", "modern, korean", true, "google-fonts"},
	{"Noto Sans TC", "Noto Sans TC", "cjk", "modern, traditional chinese", true, "google-fonts"},

	// ── Other open source ──
	{"Source Han Sans SC", "思源黑体 (Adobe)", "cjk", "modern, professional, chinese", true, "open-source"},
	{"Source Han Serif SC", "思源宋体 (Adobe)", "cjk", "elegant, serif, chinese", true, "open-source"},
	{"霞鹜文楷", "LXGW WenKai", "cjk", "handwriting, elegant, chinese", true, "open-source"},
}
