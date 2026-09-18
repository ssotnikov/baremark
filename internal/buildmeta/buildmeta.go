// Package buildmeta validates build-time metadata and mandatory brand assets.
package buildmeta

import (
	"encoding/binary"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const CanonicalLogoPath = "assets/app-logo.png"

var (
	CanonicalIconPaths = []string{
		"assets/icons/app/BareMark-app-light.ico",
		"assets/icons/app/BareMark-app-dark.ico",
		"assets/icons/file/BareMark-file-light.ico",
		"assets/icons/file/BareMark-file-dark.ico",
	}
	RequiredIconSizes = []int{16, 24, 32, 48, 256}
	semanticVersion   = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
)

type Version struct {
	Major uint16
	Minor uint16
	Patch uint16
	raw   string
}

func (v Version) String() string {
	return v.raw
}

type Project struct {
	Version Version
}

func ValidateProject(root string) (Project, error) {
	version, err := ReadVersion(filepath.Join(root, "version.go"))
	if err != nil {
		return Project{}, err
	}
	if err := validateLogo(filepath.Join(root, filepath.FromSlash(CanonicalLogoPath))); err != nil {
		return Project{}, err
	}
	for _, relativePath := range CanonicalIconPaths {
		path := filepath.Join(root, filepath.FromSlash(relativePath))
		if err := validateICO(path); err != nil {
			return Project{}, err
		}
	}
	return Project{Version: version}, nil
}

func ReadVersion(path string) (Version, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return Version{}, fmt.Errorf("parse version source %q: %w", path, err)
	}

	var raw string
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.CONST {
			continue
		}
		for _, specification := range general.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range value.Names {
				if name.Name != "Version" || i >= len(value.Values) {
					continue
				}
				literal, ok := value.Values[i].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return Version{}, fmt.Errorf("Version in %q must be a string literal", path)
				}
				raw, err = strconv.Unquote(literal.Value)
				if err != nil {
					return Version{}, fmt.Errorf("decode Version in %q: %w", path, err)
				}
			}
		}
	}
	if raw == "" {
		return Version{}, fmt.Errorf("Version constant not found in %q", path)
	}

	matches := semanticVersion.FindStringSubmatch(raw)
	if matches == nil {
		return Version{}, fmt.Errorf("Version %q is not a semantic version", raw)
	}
	parts := make([]uint16, 3)
	for i := range parts {
		part, parseErr := strconv.ParseUint(matches[i+1], 10, 16)
		if parseErr != nil {
			return Version{}, fmt.Errorf("Version component %q exceeds PE limit 65535", matches[i+1])
		}
		parts[i] = uint16(part)
	}
	return Version{Major: parts[0], Minor: parts[1], Patch: parts[2], raw: raw}, nil
}

func validateLogo(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open mandatory logo %q: %w", path, err)
	}
	defer file.Close()

	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return fmt.Errorf("decode mandatory logo %q: %w", path, err)
	}
	if format != "png" || config.Width <= 0 || config.Height <= 0 {
		return fmt.Errorf("mandatory logo %q must be a non-empty PNG", path)
	}
	return nil
}

func validateICO(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read mandatory icon %q: %w", path, err)
	}
	if len(data) < 6 {
		return fmt.Errorf("mandatory icon %q has a truncated ICO header", path)
	}
	if binary.LittleEndian.Uint16(data[0:2]) != 0 || binary.LittleEndian.Uint16(data[2:4]) != 1 {
		return fmt.Errorf("mandatory icon %q is not an ICO file", path)
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if len(data) < 6+count*16 {
		return fmt.Errorf("mandatory icon %q has a truncated ICO directory", path)
	}

	found := make(map[int]bool, count)
	for i := 0; i < count; i++ {
		offset := 6 + i*16
		width := icoDimension(data[offset])
		height := icoDimension(data[offset+1])
		if width != height {
			return fmt.Errorf("mandatory icon %q contains non-square entry %dx%d", path, width, height)
		}
		if found[width] {
			return fmt.Errorf("mandatory icon %q contains duplicate %dx%d entry", path, width, height)
		}
		found[width] = true
	}

	var missing []string
	for _, size := range RequiredIconSizes {
		if !found[size] {
			missing = append(missing, fmt.Sprintf("%dx%d", size, size))
		}
	}
	if len(missing) > 0 || len(found) != len(RequiredIconSizes) {
		return fmt.Errorf("mandatory icon %q must contain exactly the required sizes; missing: %s", path, strings.Join(missing, ", "))
	}
	return nil
}

func icoDimension(value byte) int {
	if value == 0 {
		return 256
	}
	return int(value)
}
