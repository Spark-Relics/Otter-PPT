package fonts

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

// TTFInfo holds parsed metadata from a TrueType/OpenType font file.
type TTFInfo struct {
	FamilyName   string // e.g. "Inter", "Noto Sans SC"
	FullName     string // e.g. "Inter Regular"
	PostScript   string // e.g. "Inter-Regular"
	SubfamilyName string // e.g. "Regular", "Bold"
	IsBold       bool
	IsItalic     bool
}

// ParseTTF reads a .ttf or .otf file and extracts font name metadata.
// It parses the 'name' table (platform 3 / Windows encoding).
func ParseTTF(path string) (*TTFInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// ── Offset table ──
	var header [12]byte
	if _, err := io.ReadFull(f, header[:]); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	// sfVersion = header[0:4] — 0x00010000 (TTF) or 'OTTO' (OTF with CFF)
	numTables := binary.BigEndian.Uint16(header[4:6])

	// ── Table directory ──
	var nameOffset, nameLength uint32
	found := false
	for i := 0; i < int(numTables); i++ {
		var entry [16]byte
		if _, err := io.ReadFull(f, entry[:]); err != nil {
			return nil, fmt.Errorf("read table entry %d: %w", i, err)
		}
		tag := string(entry[0:4])
		if tag == "name" {
			nameOffset = binary.BigEndian.Uint32(entry[8:12])
			nameLength = binary.BigEndian.Uint32(entry[12:16])
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("no 'name' table found")
	}

	// ── Name table ──
	if _, err := f.Seek(int64(nameOffset), io.SeekStart); err != nil {
		return nil, err
	}

	var nameHeader [6]byte
	if _, err := io.ReadFull(f, nameHeader[:]); err != nil {
		return nil, fmt.Errorf("read name table header: %w", err)
	}
	format := binary.BigEndian.Uint16(nameHeader[0:2])
	count := binary.BigEndian.Uint16(nameHeader[2:4])
	stringOffset := binary.BigEndian.Uint16(nameHeader[4:6])

	records := make([][12]byte, count)
	for i := 0; i < int(count); i++ {
		if _, err := io.ReadFull(f, records[i][:]); err != nil {
			return nil, fmt.Errorf("read name record %d: %w", i, err)
		}
	}

	info := &TTFInfo{}
	stringPool := make([]byte, nameLength)
	if n, err := f.ReadAt(stringPool, int64(nameOffset)+int64(stringOffset)); n < int(nameLength) || err != nil {
		// Best effort — use what we got
		if n > 0 && err == io.EOF {
			// ok
		} else if err != nil {
			// Fall back to empty strings
			if info.FamilyName == "" {
				return info, nil
			}
		}
	}

	for i := 0; i < int(count); i++ {
		rec := records[i]
		platformID := binary.BigEndian.Uint16(rec[0:2])
		encodingID := binary.BigEndian.Uint16(rec[2:4])
		nameID := binary.BigEndian.Uint16(rec[6:8])
		strLen := int(binary.BigEndian.Uint16(rec[8:10]))
		strOff := int(binary.BigEndian.Uint16(rec[10:12]))

		if strOff+strLen > len(stringPool) {
			continue
		}
		raw := stringPool[strOff : strOff+strLen]

		// Prefer platform 3 (Windows) / encoding 1 (Unicode BMP)
		if platformID != 3 {
			continue
		}
		_ = encodingID // accept all windows encodings

		value := decodeUTF16BE(raw)

		switch nameID {
		case 1: // Font Family
			if info.FamilyName == "" {
				info.FamilyName = value
			}
		case 2: // Font Subfamily
			if info.SubfamilyName == "" {
				info.SubfamilyName = value
				lower := strings.ToLower(value)
				if strings.Contains(lower, "bold") {
					info.IsBold = true
				}
				if strings.Contains(lower, "italic") || strings.Contains(lower, "oblique") {
					info.IsItalic = true
				}
			}
		case 4: // Full Name
			info.FullName = value
		case 6: // PostScript Name
			if info.PostScript == "" {
				info.PostScript = value
			}
		}
	}

	// format 1 has additional lang tag records
	_ = format

	if info.FamilyName == "" && info.FullName != "" {
		info.FamilyName = info.FullName
	}

	return info, nil
}

func decodeUTF16BE(b []byte) string {
	if len(b)%2 != 0 {
		return string(b)
	}
	runes := make([]rune, 0, len(b)/2)
	for i := 0; i < len(b); i += 2 {
		r := rune(binary.BigEndian.Uint16(b[i : i+2]))
		runes = append(runes, r)
	}
	return string(runes)
}
