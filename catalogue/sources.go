package catalogue

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// SourcesVersion is the sources.json format this code understands.
const SourcesVersion = 1

// HostedSource is the id of ROMs Chippie hosts itself; it is tried last.
const HostedSource = "chippie"

// Sources is web/sources.json (README §9.3): where each ROM can be
// downloaded.
type Sources struct {
	Version   int                   `json:"version"`
	Generated string                `json:"generated"`
	Sources   map[string]SourceInfo `json:"sources"`
	ROMs      map[string][]Location `json:"roms"` // by SHA-1
}

// SourceInfo describes one archive.
type SourceInfo struct {
	Name    string `json:"name"`
	License string `json:"license,omitempty"`
	Home    string `json:"home"`
}

// Location is one URL a ROM can be downloaded from.
type Location struct {
	Source string `json:"source"`
	URL    string `json:"url"`
}

// ParseSources parses and checks a sources manifest.
func ParseSources(data []byte) (*Sources, error) {
	var s Sources
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("%w: sources.json: %w", ErrInvalid, err)
	}

	if s.Version != SourcesVersion {
		return nil, fmt.Errorf("%w: sources.json version %d, want %d", ErrInvalid, s.Version, SourcesVersion)
	}

	for hash, locs := range s.ROMs {
		for _, l := range locs {
			if _, ok := s.Sources[l.Source]; !ok {
				return nil, fmt.Errorf("%w: sources.json: ROM %s: unknown source %q", ErrInvalid, hash, l.Source)
			}

			if !strings.HasPrefix(l.URL, "https://") && l.Source != HostedSource {
				return nil, fmt.Errorf("%w: sources.json: ROM %s: URL %q isn't https", ErrInvalid, hash, l.URL)
			}
		}
	}

	return &s, nil
}

// URLs returns where a ROM can be downloaded, best first: original sources
// in manifest order, Chippie's own hosted copies last (README §9.3).
func (s *Sources) URLs(sha1 string) []Location {
	locs := append([]Location(nil), s.ROMs[strings.ToLower(sha1)]...)
	sort.SliceStable(locs, func(i, j int) bool {
		return locs[i].Source != HostedSource && locs[j].Source == HostedSource
	})

	return locs
}

// Available reports whether a ROM can be downloaded from anywhere.
func (s *Sources) Available(sha1 string) bool { return len(s.ROMs[strings.ToLower(sha1)]) > 0 }
