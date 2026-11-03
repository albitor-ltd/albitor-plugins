package app

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"
	"unicode"

	"webapp/internal/apikit"
	"webapp/internal/identity"
	"webapp/internal/objects"
)

// Files are the files attached to a record: uploaded and downloaded by the
// browser directly against the bucket. The API authorises, then signs.
//
//	POST   /api/records/{record}/files               start an upload; returns the URL to PUT to
//	POST   /api/records/{record}/files/{file}/complete  confirm the upload; returns the file's details
//	GET    /api/records/{record}/files/{file}        a short-lived download URL (302 with ?redirect=1)
//	DELETE /api/records/{record}/files/{file}        delete one file
//
// Rename "records" to what the app calls the thing the files belong to. The
// URLs in responses are paths on the app's own origin, so the web client uses
// them as they are.
type Files struct {
	Store  objects.Store
	Policy objects.Policy

	// OwnsRecord reports whether the person owns the record, and is called
	// before anything is signed. Read the record from the data store by its
	// id and compare its owner to the verified identity; answer (false, nil)
	// for a record that does not exist, so a stranger learns nothing. The
	// key space is already per owner (the owner segment comes from the
	// identity, never the request), so this check is what stops one person
	// attaching files to another's record id, and where a shared record
	// would grant access to more than its owner.
	OwnsRecord func(r *http.Request, owner, record string) (bool, error)

	// Expires is how long a URL lives; at most objects.MaxExpiry.
	Expires time.Duration
}

// FileDetails is what the app keeps in the record for each file.
type FileDetails struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

// Routes registers the file routes.
func (f *Files) Routes(r *http.ServeMux) {
	r.HandleFunc("POST /api/records/{record}/files", f.start)
	r.HandleFunc("POST /api/records/{record}/files/{file}/complete", f.complete)
	r.HandleFunc("GET /api/records/{record}/files/{file}", f.download)
	r.HandleFunc("DELETE /api/records/{record}/files/{file}", f.remove)
}

// authorise resolves the key for a request, or answers and returns false. A
// record the person does not own is a 404, the same as one that is not there.
func (f *Files) authorise(w http.ResponseWriter, r *http.Request, file string) (key, owner string, ok bool) {
	owner = identity.Owner(r)
	// The URL's record id is turned into a safe key segment; the same id
	// must be passed to OwnsRecord, which compares it with the record's own.
	record := objects.Segment(r.PathValue("record"))
	k, err := objects.Key(owner, record, file)
	if err != nil {
		// Not a user error worth a message, but worth knowing: a bad owner
		// segment means the identity is not what the key space expects.
		slog.WarnContext(r.Context(), "files: not a key", "correlation_id", apikit.CorrelationIDFromContext(r.Context()), "err", err)
		apikit.WriteError(w, http.StatusNotFound, "not found")
		return "", "", false
	}
	if f.OwnsRecord == nil {
		// Fail closed: with no ownership check nothing may be signed.
		slog.ErrorContext(r.Context(), "files: OwnsRecord is not set", "correlation_id", apikit.CorrelationIDFromContext(r.Context()))
		apikit.WriteError(w, http.StatusInternalServerError, "something went wrong")
		return "", "", false
	}
	owns, err := f.OwnsRecord(r, owner, r.PathValue("record"))
	if err != nil {
		slog.ErrorContext(r.Context(), "files: ownership check failed", "correlation_id", apikit.CorrelationIDFromContext(r.Context()), "err", err)
		apikit.WriteError(w, http.StatusInternalServerError, "something went wrong")
		return "", "", false
	}
	if !owns {
		apikit.WriteError(w, http.StatusNotFound, "not found")
		return "", "", false
	}
	return k, owner, true
}

func (f *Files) expires() time.Duration {
	if f.Expires > 0 {
		return f.Expires
	}
	return 5 * time.Minute
}

func (f *Files) fail(w http.ResponseWriter, r *http.Request, what string, err error) {
	slog.ErrorContext(r.Context(), "files: "+what, "correlation_id", apikit.CorrelationIDFromContext(r.Context()), "err", err)
	if errors.Is(err, objects.ErrUnavailable) {
		apikit.WriteError(w, http.StatusServiceUnavailable, "file storage is not available")
		return
	}
	apikit.WriteError(w, http.StatusBadGateway, "file storage is unavailable: try again shortly")
}

type startRequest struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type startResponse struct {
	FileID string         `json:"file_id"`
	Name   string         `json:"name"`
	Upload objects.Signed `json:"upload"`
}

func (f *Files) start(w http.ResponseWriter, r *http.Request) {
	var in startRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&in); err != nil {
		apikit.WriteError(w, http.StatusBadRequest, "send {name, content_type, size}")
		return
	}
	// Refuse before signing: size and type are signed into the URL, so what is
	// checked here is what S3 enforces.
	if err := f.Policy.Check(in.ContentType, in.Size); err != nil {
		apikit.WriteError(w, http.StatusUnprocessableEntity, strings.TrimPrefix(err.Error(), "objects: refused: "))
		return
	}
	id := objects.NewID()
	key, _, ok := f.authorise(w, r, id)
	if !ok {
		return
	}
	name := cleanName(in.Name)
	signed, err := f.Store.PresignPut(r.Context(), key, objects.PutSpec{
		ContentType: strings.ToLower(in.ContentType), Size: in.Size, Filename: name, Expires: f.expires(),
	})
	if err != nil {
		f.fail(w, r, "presign put", err)
		return
	}
	apikit.WriteJSON(w, http.StatusCreated, startResponse{FileID: id, Name: name, Upload: signed})
}

// complete confirms the object landed and matches the policy, then returns the
// details to store on the record. Add the file to the record here, in the
// same place that checks it: an upload nobody completes is an orphan that
// the retention rules in the skill sweep.
//
// complete is not bound to the start call that issued the URL, and does not
// need to be for safety: the key sits under the caller's own owner and record
// prefix, the file id is random, and the signed length and type mean any
// object there already conforms to the policy (re-checked below). What it does
// not do is make a file single-use or cap how many a record has. So make the
// record write idempotent on the file id (a second complete of the same id
// changes nothing), and cap files per record in that write if the product needs
// a limit.
func (f *Files) complete(w http.ResponseWriter, r *http.Request) {
	key, _, ok := f.authorise(w, r, r.PathValue("file"))
	if !ok {
		return
	}
	info, err := f.Store.Head(r.Context(), key)
	if errors.Is(err, objects.ErrNotFound) {
		apikit.WriteError(w, http.StatusConflict, "the file has not been uploaded")
		return
	}
	if err != nil {
		f.fail(w, r, "head", err)
		return
	}
	if err := f.Policy.Check(info.ContentType, info.Size); err != nil {
		_ = f.Store.Delete(r.Context(), key) // never keep what the policy refuses
		apikit.WriteError(w, http.StatusUnprocessableEntity, "the uploaded file is not accepted")
		return
	}
	apikit.WriteJSON(w, http.StatusOK, FileDetails{ID: r.PathValue("file"), Name: info.Filename, ContentType: info.ContentType, Size: info.Size})
}

func (f *Files) download(w http.ResponseWriter, r *http.Request) {
	key, _, ok := f.authorise(w, r, r.PathValue("file"))
	if !ok {
		return
	}
	_, err := f.Store.Head(r.Context(), key)
	if errors.Is(err, objects.ErrNotFound) {
		apikit.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		f.fail(w, r, "head", err)
		return
	}
	signed, err := f.Store.PresignGet(r.Context(), key, f.expires())
	if err != nil {
		f.fail(w, r, "presign get", err)
		return
	}
	// ?redirect=1 lets a plain link or <img src> work with cookie sessions.
	if r.URL.Query().Get("redirect") == "1" {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		http.Redirect(w, r, signed.URL, http.StatusFound)
		return
	}
	apikit.WriteJSON(w, http.StatusOK, signed)
}

func (f *Files) remove(w http.ResponseWriter, r *http.Request) {
	key, _, ok := f.authorise(w, r, r.PathValue("file"))
	if !ok {
		return
	}
	if err := f.Store.Delete(r.Context(), key); err != nil {
		f.fail(w, r, "delete", err)
		return
	}
	// Remove the file from the record's list in the same handler.
	w.WriteHeader(http.StatusNoContent)
}

// DeleteRecordFiles removes every file a record owns. The bucket is versioned:
// each delete leaves a delete marker and the earlier versions stay in the bucket
// until the platform's noncurrent-version expiry removes them, so "deleted" means
// gone from the app, not instantly gone from storage. Call it from the
// handler that deletes the record, before deleting the record itself: a
// failure then leaves the record to retry from, where deleting the record
// first would leave files nothing points at.
func DeleteRecordFiles(r *http.Request, s objects.Store, record string) error {
	prefix, err := objects.Prefix(identity.Owner(r), objects.Segment(record))
	if err != nil {
		return err
	}
	return objects.DeleteUnder(r.Context(), s, prefix)
}

// cleanName is a person's file name made safe to keep and show: the base
// name only, no control characters, at most 200 bytes. It is never used in a
// key.
func cleanName(name string) string {
	name = path.Base(strings.ReplaceAll(name, `\`, "/"))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
	for len(name) > 200 {
		name = name[:len(name)-1]
	}
	name = strings.ToValidUTF8(name, "")
	if name == "" || name == "." || name == ".." || name == "/" {
		return "file"
	}
	return name
}
