package gh

import (
	"encoding/json"
	"testing"
)

const samplePage = `{"data":{"repositoryOwner":{"projectV2":{
  "id":"PVT_1","title":"[BOARD] Plataforma","url":"https://github.com/orgs/acme/projects/12","public":false,
  "fields":{"nodes":[
    {},
    {"id":"F_STATUS","name":"Status","options":[{"id":"o_todo","name":"Todo","description":"","color":"GRAY"},{"id":"o_prog","name":"In Progress","description":"","color":"YELLOW"}]},
    {"id":"F_TIPO","name":"tipo-da-tarefa","options":[{"id":"t_task","name":"Task","description":"","color":"BLUE"},{"id":"t_bug","name":"Bug","description":"","color":"RED"}]}
  ]},
  "items":{"pageInfo":{"hasNextPage":false,"endCursor":"c1"},"nodes":[
    {"id":"I_1","isArchived":false,
     "fieldValues":{"nodes":[{},{"optionId":"o_prog","field":{"id":"F_STATUS"}},{"optionId":"t_bug","field":{"id":"F_TIPO"}}]},
     "content":{"__typename":"Issue","id":"ISS_1","number":1290,"title":"Estudo","url":"u1","state":"OPEN",
       "repository":{"nameWithOwner":"acme/board"},"assignees":{"nodes":[{"login":"ana"}]},"subIssuesSummary":{"total":3,"completed":1}}},
    {"id":"I_2","isArchived":true,"fieldValues":{"nodes":[]},
     "content":{"__typename":"Issue","id":"ISS_2","number":1,"title":"velha","url":"u2","state":"CLOSED","repository":{"nameWithOwner":"acme/board"}}},
    {"id":"I_3","isArchived":false,"fieldValues":{"nodes":[]},
     "content":{"__typename":"DraftIssue","id":"DI_3","title":"rascunho","assignees":{"nodes":[]}}},
    {"id":"I_4","isArchived":false,"fieldValues":{"nodes":[{"optionId":"o_todo","field":{"id":"F_STATUS"}}]},
     "content":{"__typename":"PullRequest","id":"PR_4","number":7,"title":"pr","url":"u4","state":"OPEN","repository":{"nameWithOwner":"Acme/api"},"assignees":{"nodes":[]}}}
  ]}
}}}}`

func TestAssembleBoard(t *testing.T) {
	var page boardPage
	if err := decodeGraphQL([]byte(samplePage), &page); err != nil {
		t.Fatal(err)
	}
	p := page.Owner.Project
	b, err := assembleBoard(Board{Cards: []BoardCard{}, Repos: []string{}}, p.Fields.Nodes, p.Items.Nodes)
	if err != nil {
		t.Fatal(err)
	}
	if b.Status.ID != "F_STATUS" || len(b.Status.Options) != 2 {
		t.Errorf("status = %+v", b.Status)
	}
	if b.Type == nil || b.Type.ID != "F_TIPO" {
		t.Errorf("tipo = %+v", b.Type)
	}
	if len(b.Cards) != 3 {
		t.Fatalf("cards = %d, o arquivado deveria sair", len(b.Cards))
	}
	c := b.Cards[0]
	if c.Kind != "issue" || c.StatusID != "o_prog" || c.TypeID != "t_bug" || c.SubTotal != 3 || c.SubDone != 1 || c.Repo != "acme/board" || len(c.Assignees) != 1 {
		t.Errorf("card da issue = %+v", c)
	}
	if b.Cards[1].Kind != "draft" || b.Cards[1].StatusID != "" {
		t.Errorf("rascunho = %+v", b.Cards[1])
	}
	if b.Cards[2].Kind != "pr" || b.Cards[2].StatusID != "o_todo" {
		t.Errorf("pr = %+v", b.Cards[2])
	}
	if len(b.Repos) != 1 || b.Repos[0] != "acme/board" {
		t.Errorf("repos (só de issues) = %v", b.Repos)
	}
}

func TestAssembleBoardWithoutStatus(t *testing.T) {
	if _, err := assembleBoard(Board{}, []rawField{{ID: "F", Name: "Prioridade"}}, nil); err == nil {
		t.Error("sem campo Status deveria dar erro")
	}
}

func TestDecodeGraphQLError(t *testing.T) {
	err := decodeGraphQL([]byte(`{"data":null,"errors":[{"message":"Your token has not been granted the required scopes"}]}`), nil)
	if err == nil || err.Error() != "Your token has not been granted the required scopes" {
		t.Errorf("erro = %v", err)
	}
}

func TestParseIssueDetail(t *testing.T) {
	raw := `{"node":{"id":"ISS_1","number":1290,"title":"Estudo","body":"b","url":"u","state":"OPEN",
	  "assignees":{"nodes":[{"login":"ana"}]},
	  "parent":{"id":"P1","number":1200,"title":"Épico","state":"OPEN","url":"p"},
	  "subIssues":{"nodes":[{"id":"S1","number":1295,"title":"Mapear","state":"CLOSED","url":"s1"}]},
	  "linkedBranches":{"nodes":[{"ref":{"name":"task/1290-estudo"}},{"ref":null}]},
	  "repository":{"id":"R1","nameWithOwner":"acme/board","defaultBranchRef":{"name":"main","target":{"oid":"aaa"}},
	    "refs":{"nodes":[{"name":"dev","target":{"oid":"bbb"}},{"name":"main","target":{"oid":"aaa"}}]}}}}`
	var data rawIssueDetail
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatal(err)
	}
	d, err := parseIssueDetail(data)
	if err != nil {
		t.Fatal(err)
	}
	if d.Parent == nil || d.Parent.Number != 1200 {
		t.Errorf("mãe = %+v", d.Parent)
	}
	if d.RepoID != "R1" || d.DefaultBranch != "main" || len(d.SubIssues) != 1 || len(d.Linked) != 1 {
		t.Errorf("detalhe = %+v", d)
	}
	if len(d.Branches) != 2 || d.Branches[0].Name != "main" || d.Branches[1].Name != "dev" {
		t.Errorf("bases (padrão primeiro, sem repetir) = %+v", d.Branches)
	}
	if _, err := parseIssueDetail(rawIssueDetail{}); err == nil {
		t.Error("nó vazio deveria dar erro")
	}
}
