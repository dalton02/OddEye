package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Viewer devolve o login da conta ativa do gh.
func (r Runner) Viewer(ctx context.Context) (string, error) {
	out, err := r.run(ctx, nil, "api", "user", "--jq", ".login")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

type Repo struct {
	FullName string `json:"fullName"`
	Private  bool   `json:"private"`
	Pushed   string `json:"pushedAt"`
}

// ListRepos lista os repositórios a que o usuário tem acesso (próprios, de colaboração e das
// organizações), do push mais recente para o mais antigo.
func (r Runner) ListRepos(ctx context.Context) ([]Repo, error) {
	out, err := r.run(ctx, nil, "api", "--paginate", "--slurp",
		"user/repos?per_page=100&sort=pushed&affiliation=owner,collaborator,organization_member")
	if err != nil {
		return nil, err
	}
	var pages [][]struct {
		FullName string `json:"full_name"`
		Private  bool   `json:"private"`
		Archived bool   `json:"archived"`
		PushedAt string `json:"pushed_at"`
	}
	if err := json.Unmarshal(out, &pages); err != nil {
		return nil, fmt.Errorf("resposta inesperada da lista de repositórios: %w", err)
	}
	repos := []Repo{}
	for _, p := range pages {
		for _, x := range p {
			if !x.Archived {
				repos = append(repos, Repo{FullName: x.FullName, Private: x.Private, Pushed: x.PushedAt})
			}
		}
	}
	sort.SliceStable(repos, func(i, j int) bool { return repos[i].Pushed > repos[j].Pushed })
	return repos, nil
}

// OpenPR é uma PR aberta como o watcher a enxerga.
type OpenPR struct {
	Ref             PRRef     `json:"ref"`
	Title           string    `json:"title"`
	URL             string    `json:"url"`
	Author          string    `json:"author"`
	HeadSha         string    `json:"headSha"`
	IsDraft         bool      `json:"isDraft"`
	CreatedAt       time.Time `json:"createdAt" ts_type:"string"`
	ReviewRequested []string  `json:"reviewRequested"` // logins de usuários com review pedido
}

func (p OpenPR) Key() string { return strings.ToLower(p.Ref.String()) }

// reposPerSearch limita quantos `repo:` cabem numa busca: o GitHub recusa queries muito longas.
const reposPerSearch = 20

// searchQueries agrupa os repositórios em buscas de até reposPerSearch cada.
func searchQueries(repos []string) []string {
	var qs []string
	for i := 0; i < len(repos); i += reposPerSearch {
		end := min(i+reposPerSearch, len(repos))
		var b strings.Builder
		b.WriteString("is:pr is:open archived:false sort:created-desc")
		for _, r := range repos[i:end] {
			b.WriteString(" repo:" + r)
		}
		qs = append(qs, b.String())
	}
	return qs
}

const openPRsQuery = `query($q: String!, $cursor: String) {
  search(type: ISSUE, query: $q, first: 50, after: $cursor) {
    pageInfo { hasNextPage endCursor }
    nodes {
      ... on PullRequest {
        number title url isDraft createdAt headRefOid
        author { login }
        repository { name owner { login } }
        reviewRequests(first: 20) { nodes { requestedReviewer { ... on User { login } } } }
      }
    }
  }
}`

type searchResponse struct {
	Data struct {
		Search struct {
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
			Nodes []struct {
				Number     int       `json:"number"`
				Title      string    `json:"title"`
				URL        string    `json:"url"`
				IsDraft    bool      `json:"isDraft"`
				CreatedAt  time.Time `json:"createdAt"`
				HeadRefOid string    `json:"headRefOid"`
				Author     *struct {
					Login string `json:"login"`
				} `json:"author"`
				Repository struct {
					Name  string `json:"name"`
					Owner struct {
						Login string `json:"login"`
					} `json:"owner"`
				} `json:"repository"`
				ReviewRequests struct {
					Nodes []struct {
						RequestedReviewer struct {
							Login string `json:"login"`
						} `json:"requestedReviewer"`
					} `json:"nodes"`
				} `json:"reviewRequests"`
			} `json:"nodes"`
		} `json:"search"`
	} `json:"data"`
}

func parseSearch(out []byte) ([]OpenPR, string, error) {
	var res searchResponse
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, "", fmt.Errorf("resposta inesperada da busca de PRs: %w", err)
	}
	var prs []OpenPR
	for _, n := range res.Data.Search.Nodes {
		if n.Number == 0 {
			continue
		}
		p := OpenPR{
			Ref:   PRRef{Owner: n.Repository.Owner.Login, Repo: n.Repository.Name, Number: n.Number},
			Title: n.Title, URL: n.URL, HeadSha: n.HeadRefOid, IsDraft: n.IsDraft, CreatedAt: n.CreatedAt,
			ReviewRequested: []string{},
		}
		if n.Author != nil {
			p.Author = n.Author.Login
		}
		for _, rr := range n.ReviewRequests.Nodes {
			if rr.RequestedReviewer.Login != "" {
				p.ReviewRequested = append(p.ReviewRequested, rr.RequestedReviewer.Login)
			}
		}
		prs = append(prs, p)
	}
	cursor := ""
	if res.Data.Search.PageInfo.HasNextPage {
		cursor = res.Data.Search.PageInfo.EndCursor
	}
	return prs, cursor, nil
}

// maxPagesPerSearch evita varrer centenas de PRs abertas num repo grande a cada ciclo.
const maxPagesPerSearch = 4

// OpenPRs busca as PRs abertas dos repositórios (owner/name) com poucas chamadas GraphQL.
func (r Runner) OpenPRs(ctx context.Context, repos []string) ([]OpenPR, error) {
	var all []OpenPR
	for _, q := range searchQueries(repos) {
		cursor := ""
		for page := 0; page < maxPagesPerSearch; page++ {
			args := []string{"api", "graphql", "-f", "query=" + openPRsQuery, "-f", "q=" + q}
			if cursor != "" {
				args = append(args, "-f", "cursor="+cursor)
			}
			out, err := r.run(ctx, nil, args...)
			if err != nil {
				return nil, err
			}
			prs, next, err := parseSearch(out)
			if err != nil {
				return nil, err
			}
			all = append(all, prs...)
			if next == "" {
				break
			}
			cursor = next
		}
	}
	return all, nil
}
