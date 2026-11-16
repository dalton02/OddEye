package gh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Este arquivo conversa com o GitHub Projects (v2) pela API GraphQL, para a aba Board: listar os
// projetos de um dono, montar o kanban pelo campo Status e criar cards, sub-issues e branches.

// maxBoardPages limita a leitura a 1000 itens (10 páginas de 100): boards maiores ficam truncados.
const maxBoardPages = 10

type graphqlError struct {
	Message string `json:"message"`
}

// graphql roda uma query com variáveis e decodifica `data` em out. O cabeçalho de sub-issues é
// inofensivo onde o recurso já é estável e necessário onde ainda não é.
func (r Runner) graphql(ctx context.Context, query string, vars map[string]any, out any) error {
	if vars == nil {
		vars = map[string]any{}
	}
	body, err := json.Marshal(map[string]any{"query": query, "variables": vars})
	if err != nil {
		return err
	}
	raw, err := r.run(ctx, body, "api", "graphql", "-H", "GraphQL-Features: sub_issues", "--input", "-")
	if err != nil {
		return err
	}
	return decodeGraphQL(raw, out)
}

func decodeGraphQL(raw []byte, out any) error {
	var resp struct {
		Data   json.RawMessage `json:"data"`
		Errors []graphqlError  `json:"errors"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return fmt.Errorf("resposta inesperada do GitHub: %w", err)
	}
	if len(resp.Errors) > 0 {
		return errors.New(resp.Errors[0].Message)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(resp.Data, out)
}

// ProjectOwners devolve o login do usuário e das organizações dele: os donos possíveis de um projeto.
func (r Runner) ProjectOwners(ctx context.Context) ([]string, error) {
	var data struct {
		Viewer struct {
			Login         string `json:"login"`
			Organizations struct {
				Nodes []struct {
					Login string `json:"login"`
				} `json:"nodes"`
			} `json:"organizations"`
		} `json:"viewer"`
	}
	err := r.graphql(ctx, `query{viewer{login organizations(first:100){nodes{login}}}}`, nil, &data)
	if err != nil {
		return nil, err
	}
	owners := []string{}
	for _, o := range data.Viewer.Organizations.Nodes {
		owners = append(owners, o.Login)
	}
	return append(owners, data.Viewer.Login), nil
}

type ProjectSummary struct {
	ID        string `json:"id"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	Public    bool   `json:"public"`
	UpdatedAt string `json:"updatedAt"`
	Items     int    `json:"items"`
}

// ListProjects lista os projetos abertos de um usuário ou organização, do mais recente ao mais antigo.
func (r Runner) ListProjects(ctx context.Context, owner string) ([]ProjectSummary, error) {
	var data struct {
		Owner *struct {
			ProjectsV2 struct {
				Nodes []struct {
					ID        string `json:"id"`
					Number    int    `json:"number"`
					Title     string `json:"title"`
					URL       string `json:"url"`
					Public    bool   `json:"public"`
					Closed    bool   `json:"closed"`
					UpdatedAt string `json:"updatedAt"`
					Items     struct {
						TotalCount int `json:"totalCount"`
					} `json:"items"`
				} `json:"nodes"`
			} `json:"projectsV2"`
		} `json:"repositoryOwner"`
	}
	q := `query($login:String!){repositoryOwner(login:$login){... on ProjectV2Owner{
		projectsV2(first:50, orderBy:{field:UPDATED_AT, direction:DESC}){nodes{
			id number title url public closed updatedAt items{totalCount}}}}}}`
	if err := r.graphql(ctx, q, map[string]any{"login": owner}, &data); err != nil {
		return nil, err
	}
	if data.Owner == nil {
		return nil, fmt.Errorf("dono %q não encontrado", owner)
	}
	list := []ProjectSummary{}
	for _, p := range data.Owner.ProjectsV2.Nodes {
		if p.Closed {
			continue
		}
		list = append(list, ProjectSummary{ID: p.ID, Number: p.Number, Title: p.Title, URL: p.URL, Public: p.Public, UpdatedAt: p.UpdatedAt, Items: p.Items.TotalCount})
	}
	return list, nil
}

type BoardOption struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

// BoardField é um campo de seleção única do projeto (Status, tipo da tarefa).
type BoardField struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Options []BoardOption `json:"options"`
}

type BoardCard struct {
	ItemID    string   `json:"itemId"`
	Kind      string   `json:"kind"`      // issue, pr ou draft
	ContentID string   `json:"contentId"` // id do nó da issue/PR (vazio no rascunho)
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	URL       string   `json:"url"`
	State     string   `json:"state"`
	Repo      string   `json:"repo"` // owner/name
	Assignees []string `json:"assignees"`
	StatusID  string   `json:"statusId"` // opção do campo Status; vazio = sem status
	TypeID    string   `json:"typeId"`   // opção do campo de tipo; vazio = sem tipo
	SubTotal  int      `json:"subTotal"`
	SubDone   int      `json:"subDone"`
}

type Board struct {
	ProjectID string      `json:"projectId"`
	Owner     string      `json:"owner"`
	Number    int         `json:"number"`
	Title     string      `json:"title"`
	URL       string      `json:"url"`
	Public    bool        `json:"public"`
	Status    BoardField  `json:"status"`
	Type      *BoardField `json:"type"` // nil quando o projeto não tem campo de tipo
	Cards     []BoardCard `json:"cards"`
	Repos     []string    `json:"repos"`     // repositórios das issues do board, para criar cards
	Truncated bool        `json:"truncated"` // o board tem mais itens que o limite lido
}

type rawField struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Options []BoardOption `json:"options"`
}

type rawItem struct {
	ID          string `json:"id"`
	IsArchived  bool   `json:"isArchived"`
	FieldValues struct {
		Nodes []struct {
			OptionID string `json:"optionId"`
			Field    struct {
				ID string `json:"id"`
			} `json:"field"`
		} `json:"nodes"`
	} `json:"fieldValues"`
	Content *struct {
		Typename   string `json:"__typename"`
		ID         string `json:"id"`
		Number     int    `json:"number"`
		Title      string `json:"title"`
		URL        string `json:"url"`
		State      string `json:"state"`
		Repository *struct {
			NameWithOwner string `json:"nameWithOwner"`
		} `json:"repository"`
		Assignees *struct {
			Nodes []struct {
				Login string `json:"login"`
			} `json:"nodes"`
		} `json:"assignees"`
		SubIssuesSummary *struct {
			Total     int `json:"total"`
			Completed int `json:"completed"`
		} `json:"subIssuesSummary"`
	} `json:"content"`
}

type boardPage struct {
	Owner *struct {
		Project *struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			URL    string `json:"url"`
			Public bool   `json:"public"`
			Fields struct {
				Nodes []rawField `json:"nodes"`
			} `json:"fields"`
			Items struct {
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Nodes []rawItem `json:"nodes"`
			} `json:"items"`
		} `json:"projectV2"`
	} `json:"repositoryOwner"`
}

const boardQuery = `query($login:String!, $number:Int!, $after:String){
  repositoryOwner(login:$login){... on ProjectV2Owner{ projectV2(number:$number){
    id title url public
    fields(first:50){nodes{... on ProjectV2SingleSelectField{id name options{id name description color}}}}
    items(first:100, after:$after){
      pageInfo{hasNextPage endCursor}
      nodes{
        id isArchived
        fieldValues(first:30){nodes{... on ProjectV2ItemFieldSingleSelectValue{optionId field{... on ProjectV2SingleSelectField{id}}}}}
        content{
          __typename
          ... on Issue{id number title url state repository{nameWithOwner} assignees(first:5){nodes{login}} subIssuesSummary{total completed}}
          ... on PullRequest{id number title url state repository{nameWithOwner} assignees(first:5){nodes{login}}}
          ... on DraftIssue{id title assignees(first:5){nodes{login}}}
        }
      }
    }
  }}}
}`

// Board lê o projeto inteiro (até o limite de páginas) e monta o kanban pelo campo Status.
func (r Runner) Board(ctx context.Context, owner string, number int) (Board, error) {
	b := Board{Owner: owner, Number: number, Cards: []BoardCard{}, Repos: []string{}}
	var items []rawItem
	var fields []rawField
	after := ""
	for page := 0; ; page++ {
		vars := map[string]any{"login": owner, "number": number}
		if after != "" {
			vars["after"] = after
		}
		var data boardPage
		if err := r.graphql(ctx, boardQuery, vars, &data); err != nil {
			return b, err
		}
		if data.Owner == nil || data.Owner.Project == nil {
			return b, fmt.Errorf("projeto %s #%d não encontrado (falta o escopo project no gh?)", owner, number)
		}
		p := data.Owner.Project
		if page == 0 {
			b.ProjectID, b.Title, b.URL, b.Public = p.ID, p.Title, p.URL, p.Public
			fields = p.Fields.Nodes
		}
		items = append(items, p.Items.Nodes...)
		if !p.Items.PageInfo.HasNextPage {
			break
		}
		if page+1 >= maxBoardPages {
			b.Truncated = true
			break
		}
		after = p.Items.PageInfo.EndCursor
	}
	return assembleBoard(b, fields, items)
}

var typeFieldName = regexp.MustCompile(`(?i)tipo|type`)

// assembleBoard escolhe os campos (Status e o de tipo) e converte os itens em cards.
func assembleBoard(b Board, fields []rawField, items []rawItem) (Board, error) {
	var status, kind *BoardField
	for _, f := range fields {
		if f.ID == "" {
			continue // campo que não é de seleção única
		}
		bf := BoardField{ID: f.ID, Name: f.Name, Options: f.Options}
		if bf.Options == nil {
			bf.Options = []BoardOption{}
		}
		switch {
		case status == nil && strings.EqualFold(f.Name, "Status"):
			status = &bf
		case kind == nil && typeFieldName.MatchString(f.Name):
			kind = &bf
		}
	}
	if status == nil {
		return b, errors.New("o projeto não tem um campo Status de seleção única")
	}
	b.Status, b.Type = *status, kind

	repos := map[string]bool{}
	for _, it := range items {
		if it.IsArchived || it.Content == nil {
			continue
		}
		c := it.Content
		card := BoardCard{ItemID: it.ID, ContentID: c.ID, Number: c.Number, Title: c.Title, URL: c.URL, State: c.State, Assignees: []string{}}
		switch c.Typename {
		case "Issue":
			card.Kind = "issue"
		case "PullRequest":
			card.Kind = "pr"
		default:
			card.Kind = "draft"
		}
		if c.Repository != nil {
			card.Repo = c.Repository.NameWithOwner
			if card.Kind == "issue" {
				repos[card.Repo] = true
			}
		}
		if c.Assignees != nil {
			for _, a := range c.Assignees.Nodes {
				card.Assignees = append(card.Assignees, a.Login)
			}
		}
		if c.SubIssuesSummary != nil {
			card.SubTotal, card.SubDone = c.SubIssuesSummary.Total, c.SubIssuesSummary.Completed
		}
		for _, v := range it.FieldValues.Nodes {
			switch {
			case v.Field.ID == "":
			case v.Field.ID == status.ID:
				card.StatusID = v.OptionID
			case kind != nil && v.Field.ID == kind.ID:
				card.TypeID = v.OptionID
			}
		}
		b.Cards = append(b.Cards, card)
	}
	for r := range repos {
		b.Repos = append(b.Repos, r)
	}
	slices.SortFunc(b.Repos, func(x, y string) int { return strings.Compare(strings.ToLower(x), strings.ToLower(y)) })
	return b, nil
}

// SetItemField muda um campo de seleção única do item; optionID vazio limpa o campo.
func (r Runner) SetItemField(ctx context.Context, projectID, itemID, fieldID, optionID string) error {
	vars := map[string]any{"p": projectID, "i": itemID, "f": fieldID}
	if optionID == "" {
		return r.graphql(ctx, `mutation($p:ID!,$i:ID!,$f:ID!){clearProjectV2ItemFieldValue(input:{projectId:$p,itemId:$i,fieldId:$f}){clientMutationId}}`, vars, nil)
	}
	vars["o"] = optionID
	return r.graphql(ctx, `mutation($p:ID!,$i:ID!,$f:ID!,$o:String!){updateProjectV2ItemFieldValue(input:{projectId:$p,itemId:$i,fieldId:$f,value:{singleSelectOptionId:$o}}){clientMutationId}}`, vars, nil)
}

type repoInfo struct {
	ID       string
	ViewerID string
}

func (r Runner) repoInfo(ctx context.Context, repo string) (repoInfo, error) {
	owner, name, ok := strings.Cut(repo, "/")
	if !ok {
		return repoInfo{}, fmt.Errorf("repositório inválido: %q", repo)
	}
	var data struct {
		Viewer struct {
			ID string `json:"id"`
		} `json:"viewer"`
		Repository *struct {
			ID string `json:"id"`
		} `json:"repository"`
	}
	q := `query($o:String!,$n:String!){viewer{id} repository(owner:$o,name:$n){id}}`
	if err := r.graphql(ctx, q, map[string]any{"o": owner, "n": name}, &data); err != nil {
		return repoInfo{}, err
	}
	if data.Repository == nil {
		return repoInfo{}, fmt.Errorf("repositório %s não encontrado", repo)
	}
	return repoInfo{ID: data.Repository.ID, ViewerID: data.Viewer.ID}, nil
}

type createdIssue struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	URL    string `json:"url"`
	State  string `json:"state"`
}

func (r Runner) createIssue(ctx context.Context, repoID, title, body string, assignees []string) (createdIssue, error) {
	var data struct {
		CreateIssue struct {
			Issue createdIssue `json:"issue"`
		} `json:"createIssue"`
	}
	vars := map[string]any{"r": repoID, "t": title, "b": body, "a": assignees}
	q := `mutation($r:ID!,$t:String!,$b:String,$a:[ID!]){createIssue(input:{repositoryId:$r,title:$t,body:$b,assigneeIds:$a}){issue{id number url state}}}`
	err := r.graphql(ctx, q, vars, &data)
	return data.CreateIssue.Issue, err
}

func (r Runner) addToProject(ctx context.Context, projectID, contentID string) (string, error) {
	var data struct {
		Add struct {
			Item struct {
				ID string `json:"id"`
			} `json:"item"`
		} `json:"addProjectV2ItemById"`
	}
	q := `mutation($p:ID!,$c:ID!){addProjectV2ItemById(input:{projectId:$p,contentId:$c}){item{id}}}`
	err := r.graphql(ctx, q, map[string]any{"p": projectID, "c": contentID}, &data)
	return data.Add.Item.ID, err
}

// NewCard é o que a criação rápida manda: a issue, o projeto e os valores de campo da coluna.
type NewCard struct {
	ProjectID      string `json:"projectId"`
	Repo           string `json:"repo"`
	Title          string `json:"title"`
	Body           string `json:"body"`
	AssignMe       bool   `json:"assignMe"`
	StatusFieldID  string `json:"statusFieldId"`
	StatusOptionID string `json:"statusOptionId"`
	TypeFieldID    string `json:"typeFieldId"`
	TypeOptionID   string `json:"typeOptionId"`
}

// CreatedCard traz o card novo e, se um passo depois da issue falhou, o aviso do que faltou.
type CreatedCard struct {
	Card    BoardCard `json:"card"`
	Warning string    `json:"warning"`
}

// CreateCard cria a issue, põe no projeto e preenche Status e tipo. A branch, quando pedida, o
// frontend cria depois com CreateBranch, porque o nome dela leva o número da issue.
// A issue já existe a partir do segundo passo: falhas dali em diante viram aviso, não erro.
func (r Runner) CreateCard(ctx context.Context, in NewCard, viewer string) (CreatedCard, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return CreatedCard{}, errors.New("o card precisa de um título")
	}
	info, err := r.repoInfo(ctx, in.Repo)
	if err != nil {
		return CreatedCard{}, err
	}
	assignees := []string{}
	if in.AssignMe {
		assignees = append(assignees, info.ViewerID)
	}
	issue, err := r.createIssue(ctx, info.ID, title, in.Body, assignees)
	if err != nil {
		return CreatedCard{}, err
	}
	card := BoardCard{Kind: "issue", ContentID: issue.ID, Number: issue.Number, Title: title, URL: issue.URL, State: issue.State, Repo: in.Repo, Assignees: []string{}}
	if in.AssignMe && viewer != "" {
		card.Assignees = append(card.Assignees, viewer)
	}
	warn := func(step string, err error) CreatedCard {
		return CreatedCard{Card: card, Warning: fmt.Sprintf("issue #%d criada, mas %s: %v", issue.Number, step, err)}
	}
	if card.ItemID, err = r.addToProject(ctx, in.ProjectID, issue.ID); err != nil {
		return warn("não entrou no projeto", err), nil
	}
	if in.StatusFieldID != "" && in.StatusOptionID != "" {
		if err := r.SetItemField(ctx, in.ProjectID, card.ItemID, in.StatusFieldID, in.StatusOptionID); err != nil {
			return warn("o Status não foi definido", err), nil
		}
		card.StatusID = in.StatusOptionID
	}
	if in.TypeFieldID != "" && in.TypeOptionID != "" {
		if err := r.SetItemField(ctx, in.ProjectID, card.ItemID, in.TypeFieldID, in.TypeOptionID); err != nil {
			return warn("o tipo não foi definido", err), nil
		}
		card.TypeID = in.TypeOptionID
	}
	return CreatedCard{Card: card}, nil
}

type SubIssue struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	URL    string `json:"url"`
}

type BranchRef struct {
	Name string `json:"name"`
	Oid  string `json:"oid"`
}

type IssueDetail struct {
	ID            string      `json:"id"`
	Number        int         `json:"number"`
	Title         string      `json:"title"`
	Body          string      `json:"body"`
	URL           string      `json:"url"`
	State         string      `json:"state"`
	Repo          string      `json:"repo"`
	RepoID        string      `json:"repoId"`
	DefaultBranch string      `json:"defaultBranch"`
	Assignees     []string    `json:"assignees"`
	Parent        *SubIssue   `json:"parent"` // a issue mãe, quando esta é uma sub-issue
	SubIssues     []SubIssue  `json:"subIssues"`
	Linked        []string    `json:"linked"`   // branches ligadas à issue
	Branches      []BranchRef `json:"branches"` // bases possíveis: a padrão primeiro, depois as mais recentes
}

type rawIssueDetail struct {
	Node *struct {
		ID        string `json:"id"`
		Number    int    `json:"number"`
		Title     string `json:"title"`
		Body      string `json:"body"`
		URL       string `json:"url"`
		State     string `json:"state"`
		Assignees struct {
			Nodes []struct {
				Login string `json:"login"`
			} `json:"nodes"`
		} `json:"assignees"`
		Parent    *SubIssue `json:"parent"`
		SubIssues struct {
			Nodes []SubIssue `json:"nodes"`
		} `json:"subIssues"`
		LinkedBranches struct {
			Nodes []struct {
				Ref *struct {
					Name string `json:"name"`
				} `json:"ref"`
			} `json:"nodes"`
		} `json:"linkedBranches"`
		Repository *struct {
			ID               string `json:"id"`
			NameWithOwner    string `json:"nameWithOwner"`
			DefaultBranchRef *struct {
				Name   string `json:"name"`
				Target struct {
					Oid string `json:"oid"`
				} `json:"target"`
			} `json:"defaultBranchRef"`
			Refs struct {
				Nodes []struct {
					Name   string `json:"name"`
					Target struct {
						Oid string `json:"oid"`
					} `json:"target"`
				} `json:"nodes"`
			} `json:"refs"`
		} `json:"repository"`
	} `json:"node"`
}

// Issue busca o que o painel do card mostra: corpo, sub-issues, branches ligadas e as bases possíveis.
func (r Runner) Issue(ctx context.Context, id string) (IssueDetail, error) {
	q := `query($id:ID!){node(id:$id){... on Issue{
		id number title body url state
		assignees(first:10){nodes{login}}
		parent{id number title state url}
		subIssues(first:50){nodes{id number title state url}}
		linkedBranches(first:20){nodes{ref{name}}}
		repository{id nameWithOwner defaultBranchRef{name target{oid}}
			refs(refPrefix:"refs/heads/", first:30, orderBy:{field:TAG_COMMIT_DATE, direction:DESC}){nodes{name target{oid}}}}
	}}}`
	var data rawIssueDetail
	if err := r.graphql(ctx, q, map[string]any{"id": id}, &data); err != nil {
		return IssueDetail{}, err
	}
	return parseIssueDetail(data)
}

func parseIssueDetail(data rawIssueDetail) (IssueDetail, error) {
	n := data.Node
	if n == nil || n.ID == "" || n.Repository == nil {
		return IssueDetail{}, errors.New("issue não encontrada")
	}
	d := IssueDetail{ID: n.ID, Number: n.Number, Title: n.Title, Body: n.Body, URL: n.URL, State: n.State,
		Repo: n.Repository.NameWithOwner, RepoID: n.Repository.ID,
		Assignees: []string{}, Parent: n.Parent, SubIssues: n.SubIssues.Nodes, Linked: []string{}, Branches: []BranchRef{}}
	if d.SubIssues == nil {
		d.SubIssues = []SubIssue{}
	}
	for _, a := range n.Assignees.Nodes {
		d.Assignees = append(d.Assignees, a.Login)
	}
	for _, l := range n.LinkedBranches.Nodes {
		if l.Ref != nil {
			d.Linked = append(d.Linked, l.Ref.Name)
		}
	}
	if def := n.Repository.DefaultBranchRef; def != nil {
		d.DefaultBranch = def.Name
		d.Branches = append(d.Branches, BranchRef{Name: def.Name, Oid: def.Target.Oid})
	}
	for _, ref := range n.Repository.Refs.Nodes {
		if ref.Name != d.DefaultBranch {
			d.Branches = append(d.Branches, BranchRef{Name: ref.Name, Oid: ref.Target.Oid})
		}
	}
	return d, nil
}

// CreateSubIssue cria a issue no repositório da mãe, liga como sub-issue e põe no projeto com o
// Status pedido (vazio = sem status).
func (r Runner) CreateSubIssue(ctx context.Context, projectID, parentID, repo, title, statusFieldID, statusOptionID string) (SubIssue, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return SubIssue{}, errors.New("a sub-issue precisa de um título")
	}
	info, err := r.repoInfo(ctx, repo)
	if err != nil {
		return SubIssue{}, err
	}
	issue, err := r.createIssue(ctx, info.ID, title, "", nil)
	if err != nil {
		return SubIssue{}, err
	}
	sub := SubIssue{ID: issue.ID, Number: issue.Number, Title: title, State: issue.State, URL: issue.URL}
	q := `mutation($p:ID!,$s:ID!){addSubIssue(input:{issueId:$p,subIssueId:$s}){issue{id}}}`
	if err := r.graphql(ctx, q, map[string]any{"p": parentID, "s": issue.ID}, nil); err != nil {
		return sub, fmt.Errorf("issue #%d criada, mas não virou sub-issue: %w", issue.Number, err)
	}
	if projectID == "" {
		return sub, nil
	}
	itemID, err := r.addToProject(ctx, projectID, issue.ID)
	if err != nil {
		return sub, fmt.Errorf("sub-issue #%d criada, mas não entrou no projeto: %w", issue.Number, err)
	}
	if statusFieldID != "" && statusOptionID != "" {
		if err := r.SetItemField(ctx, projectID, itemID, statusFieldID, statusOptionID); err != nil {
			return sub, fmt.Errorf("sub-issue #%d criada, mas o Status não foi definido: %w", issue.Number, err)
		}
	}
	return sub, nil
}

// SetIssueClosed fecha (como concluída) ou reabre uma issue.
func (r Runner) SetIssueClosed(ctx context.Context, id string, closed bool) error {
	q := `mutation($id:ID!){reopenIssue(input:{issueId:$id}){issue{id}}}`
	if closed {
		q = `mutation($id:ID!){closeIssue(input:{issueId:$id, stateReason:COMPLETED}){issue{id}}}`
	}
	return r.graphql(ctx, q, map[string]any{"id": id}, nil)
}

// SetAssignedToMe atribui a issue à conta do gh ou tira a atribuição dela.
func (r Runner) SetAssignedToMe(ctx context.Context, id string, assign bool) error {
	var viewer struct {
		Viewer struct {
			ID string `json:"id"`
		} `json:"viewer"`
	}
	if err := r.graphql(ctx, `query{viewer{id}}`, nil, &viewer); err != nil {
		return err
	}
	q := `mutation($id:ID!,$u:[ID!]!){removeAssigneesFromAssignable(input:{assignableId:$id,assigneeIds:$u}){clientMutationId}}`
	if assign {
		q = `mutation($id:ID!,$u:[ID!]!){addAssigneesToAssignable(input:{assignableId:$id,assigneeIds:$u}){clientMutationId}}`
	}
	return r.graphql(ctx, q, map[string]any{"id": id, "u": []string{viewer.Viewer.ID}}, nil)
}

// CreateBranch cria a branch remota a partir do commit base e liga à issue ("Development" no GitHub).
func (r Runner) CreateBranch(ctx context.Context, issueID, repoID, name, baseOid string) (string, error) {
	name = strings.TrimPrefix(strings.TrimSpace(name), "refs/heads/")
	if name == "" {
		return "", errors.New("a branch precisa de um nome")
	}
	var data struct {
		Create struct {
			LinkedBranch *struct {
				Ref *struct {
					Name string `json:"name"`
				} `json:"ref"`
			} `json:"linkedBranch"`
		} `json:"createLinkedBranch"`
	}
	q := `mutation($i:ID!,$r:ID!,$n:String!,$o:GitObjectID!){createLinkedBranch(input:{issueId:$i,repositoryId:$r,name:$n,oid:$o}){linkedBranch{ref{name}}}}`
	if err := r.graphql(ctx, q, map[string]any{"i": issueID, "r": repoID, "n": name, "o": baseOid}, &data); err != nil {
		return "", err
	}
	if data.Create.LinkedBranch == nil || data.Create.LinkedBranch.Ref == nil {
		return "", errors.New("o GitHub não devolveu a branch criada (já existe uma com esse nome?)")
	}
	return data.Create.LinkedBranch.Ref.Name, nil
}
