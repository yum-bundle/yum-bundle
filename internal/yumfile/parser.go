package yumfile

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// EntryType represents the kind of directive in a Yumfile line.
type EntryType string

const (
	// EntryTypeYum installs a package via yum/dnf.
	EntryTypeYum EntryType = "yum"
	// EntryTypeKey imports a GPG key via rpm --import.
	EntryTypeKey EntryType = "key"
	// EntryTypeRepo downloads a .repo file from a URL into /etc/yum.repos.d/.
	EntryTypeRepo EntryType = "repo"
	// EntryTypeBaseurl creates a minimal .repo file from a baseurl.
	EntryTypeBaseurl EntryType = "baseurl"
	// EntryTypeCopr enables a COPR repository (Fedora community repos).
	EntryTypeCopr EntryType = "copr"
	// EntryTypeEPEL enables EPEL (Extra Packages for Enterprise Linux).
	EntryTypeEPEL EntryType = "epel"
	// EntryTypeModule enables a DNF module stream (RHEL8+/Fedora).
	EntryTypeModule EntryType = "module"
	// EntryTypeRPM installs an RPM package directly from a URL.
	EntryTypeRPM EntryType = "rpm"
	// EntryTypeGroup installs a package group via yum/dnf groupinstall.
	EntryTypeGroup EntryType = "group"
	// EntryTypeExclude prevents a package from being installed or upgraded.
	// The value is passed as --exclude=<pkg> to every dnf/yum install/groupinstall.
	EntryTypeExclude EntryType = "exclude"
)

// Entry represents a single parsed directive from a Yumfile, including
// the directive type, its argument value, the source line number, and
// the original unparsed line text.
type Entry struct {
	Type         EntryType
	Value        string
	LineNum      int
	Original     string
	ChecksumAlgo string // "sha256", "sha512", or "" for none
	Checksum     string // lowercase hex string, or "" for none
}

// Parse reads a Yumfile at the given path and returns the list of entries.
// Blank lines and comment lines (starting with #) are skipped.
func Parse(filePath string) ([]Entry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open Yumfile: %w", err)
	}
	defer file.Close()

	var entries []Entry
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		original := line

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		// Strip inline comment: everything from the first " #" to end of line.
		// Space-hash is unambiguous since # does not appear in package names,
		// URLs, COPR identifiers, or any other directive argument.
		if idx := strings.Index(line, " #"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line == "" {
			continue
		}

		entry, err := parseLine(line, lineNum, original)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return entries, nil
}

func parseLine(line string, lineNum int, original string) (Entry, error) {
	// epel is a bare directive with no argument
	if strings.TrimSpace(line) == "epel" {
		return Entry{
			Type:     EntryTypeEPEL,
			Value:    "",
			LineNum:  lineNum,
			Original: original,
		}, nil
	}

	parts := splitRespectingQuotes(line)
	if len(parts) < 2 {
		return Entry{}, fmt.Errorf("invalid line format: expected 'directive argument'")
	}

	directive := parts[0]

	var entryType EntryType
	switch directive {
	case "yum":
		entryType = EntryTypeYum
	case "key":
		entryType = EntryTypeKey
	case "repo":
		entryType = EntryTypeRepo
	case "baseurl":
		entryType = EntryTypeBaseurl
	case "copr":
		entryType = EntryTypeCopr
	case "module":
		entryType = EntryTypeModule
	case "rpm":
		entryType = EntryTypeRPM
	case "group":
		entryType = EntryTypeGroup
	case "exclude":
		entryType = EntryTypeExclude
	default:
		return Entry{}, fmt.Errorf("unknown directive: %s", directive)
	}

	// For key, repo, and rpm directives, check for an optional checksum suffix:
	//   <directive> <url> sha256=<hex>
	//   <directive> <url> sha512=<hex>
	var checksumAlgo, checksum string
	valueParts := parts[1:]
	if (entryType == EntryTypeKey || entryType == EntryTypeRepo || entryType == EntryTypeRPM) && len(valueParts) >= 2 {
		last := valueParts[len(valueParts)-1]
		lower := strings.ToLower(last)
		var algo, hex string
		if after, ok := strings.CutPrefix(lower, "sha256="); ok {
			algo = "sha256"
			hex = after
		} else if after, ok := strings.CutPrefix(lower, "sha512="); ok {
			algo = "sha512"
			hex = after
		}
		if algo != "" {
			checksumAlgo = algo
			checksum = hex
			valueParts = valueParts[:len(valueParts)-1]
		}
	}

	value := strings.Join(valueParts, " ")
	value = unquote(value)

	return Entry{
		Type:         entryType,
		Value:        value,
		LineNum:      lineNum,
		Original:     original,
		ChecksumAlgo: checksumAlgo,
		Checksum:     checksum,
	}, nil
}

func splitRespectingQuotes(s string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)

	for _, r := range s {
		switch {
		case (r == '"' || r == '\'') && !inQuotes:
			inQuotes = true
			quoteChar = r
		case r == quoteChar && inQuotes:
			inQuotes = false
			quoteChar = 0
		case r == ' ' && !inQuotes:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// ExtractPkgName returns the package name from a yum spec.
// Handles formats: "curl", "curl-7.76.1", "curl = 7.76.1", "curl=7.76.1".
func ExtractPkgName(spec string) string {
	// Handle "name = version" (DNF equality format)
	if idx := strings.Index(spec, " = "); idx > 0 {
		return strings.TrimSpace(spec[:idx])
	}
	// Handle "name=version" (compact form)
	if idx := strings.Index(spec, "="); idx > 0 {
		return spec[:idx]
	}
	// Handle "name-version" only if the version part contains a digit
	// (to avoid stripping the trailing part of names like "python3-pip")
	// RPM naming: package name uses hyphens, version is numeric.
	// We split on the last hyphen followed by a digit.
	if idx := LastHyphenBeforeVersion(spec); idx > 0 {
		return spec[:idx]
	}
	return spec
}

// LastHyphenBeforeVersion finds the last hyphen in s that is followed by a digit,
// returning its index, or -1 if none found.
// This distinguishes "nodejs-18.0.0" (name=nodejs, version=18.0.0) from
// "python3-pip" (name=python3-pip, no version).
func LastHyphenBeforeVersion(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '-' && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {
			return i
		}
	}
	return -1
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
