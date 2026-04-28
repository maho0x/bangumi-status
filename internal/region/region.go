// Package region provides validation/normalization for ISO 3166-1 alpha-2
// country codes used as the probe REGION value.
package region

import "strings"

// codes holds every ISO 3166-1 alpha-2 country code (officially assigned),
// stored lowercase for direct map lookup. List is short and stable enough to
// embed; updating it is a one-line edit when ISO assigns a new code.
var codes = map[string]struct{}{}

func init() {
	for _, c := range allCodes {
		codes[c] = struct{}{}
	}
}

// Normalize trims whitespace and lowercases s. Returns the cleaned code and
// whether it is a valid ISO 3166-1 alpha-2 country code.
func Normalize(s string) (string, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "", false
	}
	_, ok := codes[s]
	return s, ok
}

// IsValid reports whether s is a valid ISO 3166-1 alpha-2 code (after
// trimming + lowercasing).
func IsValid(s string) bool {
	_, ok := Normalize(s)
	return ok
}

// allCodes is the canonical list of ISO 3166-1 alpha-2 codes (officially
// assigned, lowercased). Source: ISO 3166-1 (revised through ISO Online
// Browsing Platform). Update if ISO assigns a new code.
var allCodes = []string{
	"ad", "ae", "af", "ag", "ai", "al", "am", "ao", "aq", "ar",
	"as", "at", "au", "aw", "ax", "az", "ba", "bb", "bd", "be",
	"bf", "bg", "bh", "bi", "bj", "bl", "bm", "bn", "bo", "bq",
	"br", "bs", "bt", "bv", "bw", "by", "bz", "ca", "cc", "cd",
	"cf", "cg", "ch", "ci", "ck", "cl", "cm", "cn", "co", "cr",
	"cu", "cv", "cw", "cx", "cy", "cz", "de", "dj", "dk", "dm",
	"do", "dz", "ec", "ee", "eg", "eh", "er", "es", "et", "fi",
	"fj", "fk", "fm", "fo", "fr", "ga", "gb", "gd", "ge", "gf",
	"gg", "gh", "gi", "gl", "gm", "gn", "gp", "gq", "gr", "gs",
	"gt", "gu", "gw", "gy", "hk", "hm", "hn", "hr", "ht", "hu",
	"id", "ie", "il", "im", "in", "io", "iq", "ir", "is", "it",
	"je", "jm", "jo", "jp", "ke", "kg", "kh", "ki", "km", "kn",
	"kp", "kr", "kw", "ky", "kz", "la", "lb", "lc", "li", "lk",
	"lr", "ls", "lt", "lu", "lv", "ly", "ma", "mc", "md", "me",
	"mf", "mg", "mh", "mk", "ml", "mm", "mn", "mo", "mp", "mq",
	"mr", "ms", "mt", "mu", "mv", "mw", "mx", "my", "mz", "na",
	"nc", "ne", "nf", "ng", "ni", "nl", "no", "np", "nr", "nu",
	"nz", "om", "pa", "pe", "pf", "pg", "ph", "pk", "pl", "pm",
	"pn", "pr", "ps", "pt", "pw", "py", "qa", "re", "ro", "rs",
	"ru", "rw", "sa", "sb", "sc", "sd", "se", "sg", "sh", "si",
	"sj", "sk", "sl", "sm", "sn", "so", "sr", "ss", "st", "sv",
	"sx", "sy", "sz", "tc", "td", "tf", "tg", "th", "tj", "tk",
	"tl", "tm", "tn", "to", "tr", "tt", "tv", "tw", "tz", "ua",
	"ug", "um", "us", "uy", "uz", "va", "vc", "ve", "vg", "vi",
	"vn", "vu", "wf", "ws", "ye", "yt", "za", "zm", "zw",
}
