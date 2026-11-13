// Package watcher vigia de tempos em tempos as PRs abertas dos repositórios configurados e avisa
// quando chega uma PR nova.
package watcher

import (
	"strings"
	"time"

	"oddeye/internal/config"
	"oddeye/internal/gh"
)

// seenRetention é quanto tempo uma PR que sumiu da busca continua marcada como vista. A busca devolve
// só as PRs mais recentes de cada repo; sem essa folga, uma PR antiga que saísse e voltasse ao corte
// seria notificada de novo.
const seenRetention = 14 * 24 * time.Hour

// Evaluate aplica uma varredura ao estado e devolve as PRs que devem virar notificação.
//
// Regras: uma PR é nova quando passa nos filtros e ainda não foi vista. Na primeira varredura de um
// repositório recém-adicionado tudo que já está aberto é marcado como visto sem notificar, para não
// despejar dezenas de avisos de uma vez. Uma PR barrada pelo filtro (draft, review não pedido) não é
// marcada: se depois passar (saiu de draft, pediram meu review), ela notifica.
func Evaluate(st *config.State, cfg config.Config, viewer string, prs []gh.OpenPR, now time.Time) []gh.OpenPR {
	repos := map[string]config.RepoConfig{}
	for _, r := range cfg.Repos {
		repos[strings.ToLower(r.FullName())] = r
	}
	open := map[string]bool{}
	var fresh []gh.OpenPR
	for _, p := range prs {
		repoKey := strings.ToLower(p.Ref.Owner + "/" + p.Ref.Repo)
		rc, ok := repos[repoKey]
		if !ok {
			continue
		}
		open[p.Key()] = true
		if !passes(p, rc, cfg, viewer) {
			continue
		}
		if _, seen := st.Seen[p.Key()]; seen {
			continue
		}
		st.Seen[p.Key()] = config.SeenPR{HeadSha: p.HeadSha, SeenAt: now}
		if st.KnownRepos[repoKey] {
			fresh = append(fresh, p)
		}
	}

	for k := range st.KnownRepos {
		if _, ok := repos[k]; !ok {
			delete(st.KnownRepos, k)
		}
	}
	for k := range repos {
		st.KnownRepos[k] = true
	}
	for k, s := range st.Seen {
		if !open[k] && now.Sub(s.SeenAt) > seenRetention {
			delete(st.Seen, k)
		}
	}

	pending := st.Pending[:0:0]
	for _, p := range st.Pending {
		if open[p.Key()] {
			pending = append(pending, p)
		}
	}
	st.Pending = append(pending, fresh...)
	return fresh
}

func passes(p gh.OpenPR, rc config.RepoConfig, cfg config.Config, viewer string) bool {
	if p.IsDraft && !cfg.IncludeDrafts {
		return false
	}
	own := viewer != "" && strings.EqualFold(p.Author, viewer)
	if own && !cfg.NotifyOwnPRs {
		return false
	}
	if rc.OnlyReviewRequested && !own {
		for _, login := range p.ReviewRequested {
			if strings.EqualFold(login, viewer) {
				return true
			}
		}
		return false
	}
	return true
}
