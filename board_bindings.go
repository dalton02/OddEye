package main

import (
	"oddeye/internal/config"
	"oddeye/internal/gh"
)

// Métodos da aba Board: um kanban simplificado do GitHub Projects (internal/gh/project.go).

func (a *App) BoardPrefs() (config.BoardPrefs, error) { return a.store.LoadBoard() }

func (a *App) SaveBoardPrefs(p config.BoardPrefs) error { return a.store.SaveBoard(p) }

// BoardViewer é o login da conta do gh, para o filtro "Só meus" e para atribuir cards.
func (a *App) BoardViewer() (string, error) { return a.gh.Viewer(a.ctx) }

func (a *App) ProjectOwners() ([]string, error) { return a.gh.ProjectOwners(a.ctx) }

func (a *App) ListProjects(owner string) ([]gh.ProjectSummary, error) {
	return a.gh.ListProjects(a.ctx, owner)
}

func (a *App) LoadBoard(owner string, number int) (gh.Board, error) {
	return a.gh.Board(a.ctx, owner, number)
}

// SetCardField muda Status ou tipo de um card; optionID vazio limpa o campo.
func (a *App) SetCardField(projectID, itemID, fieldID, optionID string) error {
	return a.gh.SetItemField(a.ctx, projectID, itemID, fieldID, optionID)
}

// CreateCard cria a issue no repositório escolhido, já no projeto, e lembra o repositório para a próxima.
func (a *App) CreateCard(in gh.NewCard) (gh.CreatedCard, error) {
	viewer := ""
	if in.AssignMe {
		viewer, _ = a.gh.Viewer(a.ctx)
	}
	res, err := a.gh.CreateCard(a.ctx, in, viewer)
	if err == nil {
		if p, perr := a.store.LoadBoard(); perr == nil && p.LastRepo != in.Repo {
			p.LastRepo = in.Repo
			_ = a.store.SaveBoard(p)
		}
	}
	return res, err
}

func (a *App) IssueDetail(id string) (gh.IssueDetail, error) { return a.gh.Issue(a.ctx, id) }

func (a *App) CreateSubIssue(projectID, parentID, repo, title, statusFieldID, statusOptionID string) (gh.SubIssue, error) {
	return a.gh.CreateSubIssue(a.ctx, projectID, parentID, repo, title, statusFieldID, statusOptionID)
}

func (a *App) SetIssueClosed(id string, closed bool) error {
	return a.gh.SetIssueClosed(a.ctx, id, closed)
}

func (a *App) SetAssignedToMe(id string, assign bool) error {
	return a.gh.SetAssignedToMe(a.ctx, id, assign)
}

func (a *App) CreateIssueBranch(issueID, repoID, name, baseOid string) (string, error) {
	return a.gh.CreateBranch(a.ctx, issueID, repoID, name, baseOid)
}
