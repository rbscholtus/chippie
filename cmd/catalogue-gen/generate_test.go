package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"strings"
	"testing"

	"github.com/rbscholtus/chippie/catalogue"
)

func testDB(t *testing.T, hashes ...string) *catalogue.DB {
	t.Helper()

	roms := make([]string, 0, len(hashes))

	idx := make([]string, 0, len(hashes))
	for _, h := range hashes {
		roms = append(roms, `"`+h+`":{"platforms":["modernChip8"]}`)
		idx = append(idx, `"`+h+`":0`)
	}

	db, err := catalogue.Parse(catalogue.Files{
		Platforms: []byte(`[{"id":"modernChip8","defaultTickrate":12,"quirks":{}}]`),
		Quirks: []byte(`[{"id":"shift"},{"id":"memoryIncrementByX"},{"id":"memoryLeaveIUnchanged"},` +
			`{"id":"wrap"},{"id":"jump"},{"id":"vblank"},{"id":"logic"}]`),
		Programs: []byte(`[{"title":"A","roms":{` + strings.Join(roms, ",") + `}}]`),
		Hashes:   []byte(`{` + strings.Join(idx, ",") + `}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func TestGenerate(t *testing.T) {
	t.Parallel()

	known, unknown := []byte("known ROM"), []byte("unknown ROM")
	db := testDB(t, sha1Hex(known))

	trees := []tree{
		{
			archive: archives[0], // chip8Archive: roms/ only
			commit:  "c1",
			files: map[string][]byte{
				"roms/good.ch8":  known,
				"roms/other.ch8": unknown,
				"src/good.8o":    []byte("source, not a ROM"),
			},
		},
		{
			archive: archives[1], // kripod: files in subdirectories, not .txt
			commit:  "c2",
			files: map[string][]byte{
				"games/Good Game [Author].ch8": known,
				"games/Good Game.txt":          []byte("notes"),
				"LICENSE":                      []byte("licence text"),
			},
		},
	}

	res := generate(db, trees, "2026-11-09T00:00:00Z")

	locs := res.sources.ROMs[sha1Hex(known)]
	if len(locs) != 2 {
		t.Fatalf("known ROM has %d locations, want 2: %+v", len(locs), locs)
	}

	wantURL := "https://raw.githubusercontent.com/kripod/chip8-roms/c2/games/Good%20Game%20%5BAuthor%5D.ch8"
	if locs[1].URL != wantURL {
		t.Errorf("kripod URL %q, want %q", locs[1].URL, wantURL)
	}

	if _, ok := res.sources.ROMs[sha1Hex(unknown)]; ok {
		t.Error("a ROM that isn't in the database is in the manifest")
	}

	if res.archiveHashes["good"] != sha1Hex(known) || res.archiveHashes["other"] != sha1Hex(unknown) {
		t.Errorf("archive hashes %v", res.archiveHashes)
	}

	if res.stats[0].unique != 2 || res.stats[0].inDB != 1 || res.stats[1].unique != 1 {
		t.Errorf("stats %+v", res.stats)
	}

	if res.allUnique != 2 || res.allInDB != 1 {
		t.Errorf("combined %d/%d, want 2/1", res.allUnique, res.allInDB)
	}

	// The manifest it writes is one the app accepts.
	out := res.sources

	data, err := jsonMarshal(out)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := catalogue.ParseSources(data); err != nil {
		t.Errorf("generated manifest doesn't parse: %v", err)
	}

	md := report(res, db, []catalogue.Conflict{{Title: "A", Field: "tickrate", DB: "1", Archive: "2"}})
	for _, want := range []string{"| chip8Archive | 2 | 1 | 1 |", "roms/other.ch8", "tickrate"} {
		if !strings.Contains(md, want) {
			t.Errorf("report lacks %q:\n%s", want, md)
		}
	}
}

func TestUntar(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)

	for name, body := range map[string]string{
		"repo-abc/roms/a.ch8": "AAAA",
		"repo-abc/README.md":  "readme",
		"repo-abc/big.ch8":    strings.Repeat("x", maxROMFile+1),
	} {
		hdr := &tar.Header{Name: name, Mode: 0o600, Size: int64(len(body)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}

		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}

	if err := tw.WriteHeader(&tar.Header{Name: "repo-abc/roms/", Typeflag: tar.TypeDir}); err != nil {
		t.Fatal(err)
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	files, err := untar(buf.Bytes(), func(n string) bool { return strings.HasSuffix(n, ".ch8") })
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 || string(files["roms/a.ch8"]) != "AAAA" {
		t.Errorf("files = %v, want only roms/a.ch8 (oversized files skipped)", keys(files))
	}

	if _, err := untar([]byte("not gzip"), nil); !errors.Is(err, errBadTarball) {
		t.Errorf("bad data: err=%v", err)
	}
}

func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	return out
}
