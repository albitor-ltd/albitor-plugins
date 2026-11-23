package ownedscope

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"strconv"
	"strings"
)

// Provenance stamps the report this helper writes. Each row carries who wrote it (Helper), the kit
// release (Kit), a digest of this package's own source as it was compiled (Source) and a digest of
// the row itself (Digest), so the platform can see that a row came out of a run of this helper over
// this source and was not typed in or changed afterwards.
//
// THE THREAT MODEL, honestly. The app's builder owns every file of the app, this package included.
// A builder who wanted a green report could edit the helper and recompute the stamp, so the stamp
// does not stop that. It does three smaller things. A report written or edited by hand fails the
// check, because its digests do not match. A helper that is not the kit's shows as a different
// Source digest, which the platform compares with the digest of the kit release it knows. And an
// honest mistake (a report from a stale run, a half-merged file) is named instead of trusted. It is
// evidence for a reviewer, never proof.
type Provenance struct {
	Helper string `json:"helper"`
	Kit    string `json:"kit"`
	Source string `json:"source"`
	Digest string `json:"digest"`
}

// HelperName names this helper in a stamp.
const HelperName = "ownedscope"

// KitVersion is the api-go release this source ships in. It is bumped with the plugin's version
// (scripts/verify-api-go.sh checks the two agree).
const KitVersion = "0.12.1"

// sourceFiles are the files of this package that make the helper, in the order the digest reads
// them. Test files and testdata are not part of it. The platform computes the same digest over the
// same names in the app's tree, so a change here is a change to both.
var sourceFiles = []string{"nested.go", "openapi.go", "ownedscope.go", "provenance.go", "yaml.go"}

//go:embed nested.go openapi.go ownedscope.go provenance.go yaml.go
var sources embed.FS

// SourceDigest is the hex SHA-256 of the helper's own source: for each name in sourceFiles, the name,
// a newline, the byte length, a newline, the content and a newline.
func SourceDigest() string {
	h := sha256.New()
	for _, name := range sourceFiles {
		raw, err := sources.ReadFile(name)
		if err != nil {
			return ""
		}
		h.Write([]byte(name + "\n" + strconv.Itoa(len(raw)) + "\n"))
		h.Write(raw)
		h.Write([]byte("\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// rowDigest binds a row to the stamp: the hex SHA-256 of helper, kit, source, method (upper case),
// path, outcome and reason joined with NUL.
func rowDigest(kit, source, method, path, outcome, reason string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{HelperName, kit, source, strings.ToUpper(method), path, outcome, reason}, "\x00")))
	return hex.EncodeToString(sum[:])
}

func stamp(method, path, outcome, reason string) *Provenance {
	src := SourceDigest()
	return &Provenance{Helper: HelperName, Kit: KitVersion, Source: src, Digest: rowDigest(KitVersion, src, method, path, outcome, reason)}
}
