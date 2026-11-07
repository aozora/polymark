// Package store reads and writes a project folder. Markdown stays the
// readable document. A JSON sidecar holds schema version, language, and
// passage ids. Recovery files are crash copies of unsaved edits, not a
// version history.
package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aozora/polymark/internal/link"
	"github.com/aozora/polymark/internal/model"
)

const (
	projectFile = "polymark.json"
	recoveryDir = ".polymark/recovery"
	linksFile   = ".polymark/links.json"
	sidecarExt  = ".polymark.json"
)

var (
	ErrUnsafePath = errors.New("path escapes the project")
	ErrSymlink    = errors.New("refusing to write through a symlink")
	ErrName       = errors.New("invalid name")
	ErrConflict   = errors.New("file changed on disk")
	ErrExists     = errors.New("file already exists")
)

type ProjectMeta struct {
	Schema int    `json:"schema"`
	ID     string `json:"id"`
	Name   string `json:"name"`
}

type DocMeta struct {
	Schema   int             `json:"schema"`
	ID       string          `json:"id"`
	Language string          `json:"language"`
	Fidelity string          `json:"fidelity"`
	Passages []model.Passage `json:"passages,omitempty"`
}

type Recovery struct {
	Schema      int             `json:"schema"`
	DocID       string          `json:"docID"`
	RelPath     string          `json:"relPath"`
	Language    string          `json:"language"`
	Mode        string          `json:"mode"`
	Fidelity    string          `json:"fidelity"`
	Markdown    string          `json:"markdown"`
	Anchor      int             `json:"anchor"`
	Caret       int             `json:"caret"`
	Passages    []model.Passage `json:"passages,omitempty"`
	SavedDigest string          `json:"savedDigest"`
	Notice      string          `json:"notice,omitempty"`
	// Group is the link state at the checkpoint. Old journals omit it.
	Group json.RawMessage `json:"group,omitempty"`
}

type Node struct {
	Name     string
	Rel      string
	Dir      bool
	Children []Node
}

type Store struct {
	root         string
	beforeRename func(path string) error
}

func Open(root string) (*Store, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", abs)
	}
	eval, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, err
	}
	s := &Store{root: eval}
	if err := os.MkdirAll(filepath.Join(eval, recoveryDir), 0o700); err != nil {
		return nil, err
	}
	if _, err := os.Lstat(filepath.Join(eval, projectFile)); errors.Is(err, os.ErrNotExist) {
		meta := ProjectMeta{Schema: model.Schema, ID: model.NewID(), Name: filepath.Base(eval)}
		if err := s.writeJSON(projectFile, meta); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Store) Root() string { return s.root }

func (s *Store) Project() (ProjectMeta, error) {
	var meta ProjectMeta
	if err := s.readJSON(projectFile, &meta); err != nil {
		return ProjectMeta{}, err
	}
	if meta.Schema != model.Schema {
		return ProjectMeta{}, fmt.Errorf("project schema %d, want %d", meta.Schema, model.Schema)
	}
	return meta, nil
}

func ValidName(name string) error {
	if name == "" || name == "." || name == ".." {
		return ErrName
	}
	if strings.ContainsAny(name, "/\\\x00") || strings.HasPrefix(name, ".") {
		return ErrName
	}
	return nil
}

func (s *Store) CreateFolder(parent, name string) (string, error) {
	if err := ValidName(name); err != nil {
		return "", err
	}
	rel := name
	if parent != "" {
		rel = parent + "/" + name
	}
	full, err := s.safe(rel, false)
	if err != nil {
		return "", err
	}
	if err := os.Mkdir(full, 0o755); err != nil {
		return "", err
	}
	return rel, nil
}

func (s *Store) CreateDocument(folder, title string, lang model.Language) (string, *model.Session, error) {
	if _, err := model.ParseLanguage(string(lang)); err != nil {
		return "", nil, err
	}
	base := strings.TrimSpace(title)
	if base == "" {
		base = "Untitled"
	}
	base = strings.TrimSuffix(base, ".md")
	if err := ValidName(base); err != nil {
		return "", nil, err
	}
	rel, err := uniqueMarkdown(s, folder, base)
	if err != nil {
		return "", nil, err
	}
	body := "# " + base + "\n"
	session := model.OpenMarkdown(rel, body, "", lang, nil)
	if err := s.Save(session, ""); err != nil {
		return "", nil, err
	}
	return rel, session, nil
}

func uniqueMarkdown(s *Store, folder, base string) (string, error) {
	name := base + ".md"
	for n := 2; ; n++ {
		rel := name
		if folder != "" {
			rel = folder + "/" + name
		}
		full, err := s.safe(rel, false)
		if err != nil {
			return "", err
		}
		if _, err := os.Lstat(full); errors.Is(err, os.ErrNotExist) {
			return rel, nil
		}
		name = fmt.Sprintf("%s %d.md", base, n)
	}
}

func (s *Store) Tree() ([]Node, error) {
	var root []Node
	err := walk(s.root, "", &root)
	return root, err
}

func walk(dir, rel string, into *[]Node) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if name == ".polymark" || name == projectFile || strings.HasPrefix(name, ".polymark-tmp-") {
			continue
		}
		if strings.HasSuffix(name, sidecarExt) {
			continue
		}
		childRel := name
		if rel != "" {
			childRel = rel + "/" + name
		}
		full := filepath.Join(dir, name)
		info, err := os.Lstat(full)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if info.IsDir() {
			node := Node{Name: name, Rel: childRel, Dir: true}
			if err := walk(full, childRel, &node.Children); err != nil {
				return err
			}
			*into = append(*into, node)
			continue
		}
		if strings.HasSuffix(name, ".md") {
			*into = append(*into, Node{Name: name, Rel: childRel})
		}
	}
	return nil
}

type Loaded struct {
	Markdown string
	Meta     DocMeta
	Digest   string
}

func (s *Store) Load(rel string) (Loaded, error) {
	full, err := s.safe(rel, true)
	if err != nil {
		return Loaded{}, err
	}
	info, err := os.Lstat(full)
	if err != nil {
		return Loaded{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return Loaded{}, ErrSymlink
	}
	b, err := os.ReadFile(full)
	if err != nil {
		return Loaded{}, err
	}
	markdown := string(b)
	var meta DocMeta
	side := sidecarRel(rel)
	if err := s.readJSON(side, &meta); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Loaded{}, err
	}
	if meta.Schema == 0 {
		meta.Schema = model.Schema
	}
	if meta.Schema != model.Schema {
		return Loaded{}, fmt.Errorf("%s schema %d", side, meta.Schema)
	}
	if meta.Language == "" {
		meta.Language = string(model.LangEnglish)
	}
	return Loaded{Markdown: markdown, Meta: meta, Digest: digestBytes(b)}, nil
}

func (s *Store) Save(session *model.Session, expectDigest string) error {
	if session == nil {
		return model.ErrNoDocument
	}
	full, err := s.safe(session.RelPath, false)
	if err != nil {
		return err
	}
	if info, err := os.Lstat(full); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return ErrSymlink
		}
		got, err := hashFile(full)
		if err != nil {
			return err
		}
		if expectDigest != "" && got != expectDigest {
			return ErrConflict
		}
		if expectDigest == "" {
			return ErrExists
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	body := session.CommittedMarkdown()
	if err := s.writeBytes(session.RelPath, []byte(body)); err != nil {
		return err
	}
	meta := DocMeta{
		Schema:   model.Schema,
		ID:       session.ID,
		Language: string(session.Language),
		Fidelity: string(session.Fidelity),
		Passages: session.Passages(),
	}
	if err := s.writeJSON(sidecarRel(session.RelPath), meta); err != nil {
		return err
	}
	session.MarkSaved()
	return s.RemoveRecovery(session.ID)
}

func (s *Store) WriteRecovery(r Recovery) error {
	if r.Schema == 0 {
		r.Schema = model.Schema
	}
	if err := ValidName(r.DocID); err != nil {
		return err
	}
	return s.writeJSON(recoveryDir+"/"+r.DocID+".json", r)
}

func (s *Store) RemoveRecovery(id string) error {
	if err := ValidName(id); err != nil {
		return err
	}
	full, err := s.safe(recoveryDir+"/"+id+".json", false)
	if err != nil {
		return err
	}
	err = os.Remove(full)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (s *Store) Recoveries() ([]Recovery, error) {
	dir, err := s.safe(recoveryDir, true)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Recovery
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		var r Recovery
		if err := s.readJSON(recoveryDir+"/"+e.Name(), &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *Store) LoadLinks() (link.File, error) {
	var f link.File
	err := s.readJSON(linksFile, &f)
	if errors.Is(err, os.ErrNotExist) {
		return link.File{Schema: link.Schema}, nil
	}
	if err != nil {
		return link.File{}, err
	}
	if f.Schema != 0 && f.Schema != link.Schema {
		return link.File{}, fmt.Errorf("links schema %d", f.Schema)
	}
	return f, nil
}

func (s *Store) SaveLinks(f link.File) error {
	f.Schema = link.Schema
	return s.writeJSON(linksFile, f)
}

func sidecarRel(rel string) string { return rel + sidecarExt }

func (s *Store) safe(rel string, mustExist bool) (string, error) {
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." || rel == "" || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", ErrUnsafePath
	}
	full := filepath.Join(s.root, filepath.FromSlash(rel))
	parent := filepath.Dir(full)
	parentEval, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", err
	}
	if !within(s.root, parentEval) {
		return "", ErrUnsafePath
	}
	info, err := os.Lstat(full)
	if errors.Is(err, os.ErrNotExist) {
		if mustExist {
			return "", err
		}
		return full, nil
	}
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", ErrSymlink
	}
	eval, err := filepath.EvalSymlinks(full)
	if err != nil {
		return "", err
	}
	if !within(s.root, eval) {
		return "", ErrUnsafePath
	}
	return full, nil
}

func within(root, path string) bool {
	if path == root {
		return true
	}
	sep := string(os.PathSeparator)
	return strings.HasPrefix(path, root+sep)
}

func (s *Store) readJSON(rel string, dest any) error {
	full, err := s.safe(rel, true)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(full)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	return dec.Decode(dest)
}

func (s *Store) writeJSON(rel string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return s.writeBytes(rel, b)
}

func (s *Store) writeBytes(rel string, data []byte) error {
	full, err := s.safe(rel, false)
	if err != nil {
		return err
	}
	if info, err := os.Lstat(full); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return ErrSymlink
	}
	dir := filepath.Dir(full)
	f, err := os.CreateTemp(dir, ".polymark-tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		if !ok {
			os.Remove(tmp)
		}
	}()
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if s.beforeRename != nil {
		if err := s.beforeRename(full); err != nil {
			return err
		}
	}
	if err := os.Rename(tmp, full); err != nil {
		return err
	}
	ok = true
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		d.Close()
	}
	return nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func digestBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
