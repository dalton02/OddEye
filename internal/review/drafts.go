package review

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"oddeye/internal/appdir"
	"oddeye/internal/gh"
)

type Draft struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	StartLine int       `json:"startLine"`
	Line      int       `json:"line"`
	Body      string    `json:"body"`
	InDiff    bool      `json:"inDiff"` // fora do diff a API recusa comentário de linha: vai no corpo do review
	CreatedAt time.Time `json:"createdAt" ts_type:"string"`
}

// DraftStore persiste os rascunhos de uma PR em disco para sobreviverem ao app fechado.
type DraftStore struct {
	mu   sync.Mutex
	dir  string
	ref  gh.PRRef
	list []Draft
}

func NewDraftStore(ref gh.PRRef) (*DraftStore, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	s := &DraftStore{dir: filepath.Join(base, appdir.Name, "drafts"), ref: ref}
	data, err := os.ReadFile(s.file())
	if err == nil {
		_ = json.Unmarshal(data, &s.list)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return s, nil
}

func (s *DraftStore) file() string {
	return filepath.Join(s.dir, fmt.Sprintf("%s_%s_%d.json", s.ref.Owner, s.ref.Repo, s.ref.Number))
}

func (s *DraftStore) save() error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	if len(s.list) == 0 {
		err := os.Remove(s.file())
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	data, _ := json.MarshalIndent(s.list, "", "  ")
	return os.WriteFile(s.file(), data, 0o644)
}

func (s *DraftStore) List() []Draft {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]Draft{}, s.list...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Line < out[j].Line
	})
	return out
}

func (s *DraftStore) Upsert(d Draft) (Draft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d.StartLine == 0 || d.StartLine > d.Line {
		d.StartLine = d.Line
	}
	if d.ID == "" {
		b := make([]byte, 6)
		_, _ = rand.Read(b)
		d.ID = hex.EncodeToString(b)
		d.CreatedAt = time.Now()
		s.list = append(s.list, d)
	} else {
		found := false
		for i := range s.list {
			if s.list[i].ID == d.ID {
				d.CreatedAt = s.list[i].CreatedAt
				s.list[i] = d
				found = true
			}
		}
		if !found {
			return Draft{}, errors.New("rascunho não encontrado")
		}
	}
	return d, s.save()
}

func (s *DraftStore) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.list[:0]
	for _, d := range s.list {
		if d.ID != id {
			kept = append(kept, d)
		}
	}
	s.list = kept
	return s.save()
}

func (s *DraftStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.list = nil
	return s.save()
}

// BuildReview converte os rascunhos no payload da API: comentários dentro do diff viram
// comentários de linha; os de fora entram no corpo geral, citando arquivo e linha.
func BuildReview(commit, event, body string, drafts []Draft) gh.ReviewRequest {
	req := gh.ReviewRequest{CommitID: commit, Event: event, Body: body, Comments: []gh.ReviewComment{}}
	var outside []string
	for _, d := range drafts {
		if !d.InDiff {
			loc := fmt.Sprintf("L%d", d.Line)
			if d.StartLine != d.Line {
				loc = fmt.Sprintf("L%d-L%d", d.StartLine, d.Line)
			}
			outside = append(outside, fmt.Sprintf("**`%s:%s`**\n%s", d.Path, loc, d.Body))
			continue
		}
		c := gh.ReviewComment{Path: d.Path, Line: d.Line, Side: "RIGHT", Body: d.Body}
		if d.StartLine != d.Line {
			c.StartLine = d.StartLine
			c.StartSide = "RIGHT"
		}
		req.Comments = append(req.Comments, c)
	}
	if len(outside) > 0 {
		section := "### Comentários fora do diff\n\n"
		for i, o := range outside {
			if i > 0 {
				section += "\n\n"
			}
			section += o
		}
		if req.Body != "" {
			req.Body += "\n\n"
		}
		req.Body += section
	}
	return req
}
