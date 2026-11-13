package config

import "testing"

func TestNormalizeClampsIntervalAndDedupesRepos(t *testing.T) {
	c := Config{PollIntervalMinutes: 1, Repos: []RepoConfig{{Owner: "a", Name: "b"}, {Owner: "A", Name: "B"}, {Owner: "", Name: "x"}}}.Normalize()
	if c.PollIntervalMinutes != MinPollMinutes || len(c.Repos) != 1 {
		t.Errorf("Normalize = %+v", c)
	}
}

func TestPromptForPrefersRepoOverride(t *testing.T) {
	c := Config{ReviewerPrompt: "global", Repos: []RepoConfig{{Owner: "a", Name: "b", PromptOverride: "repo"}, {Owner: "a", Name: "c"}}}
	if got := c.PromptFor("A", "b"); got != "repo" {
		t.Errorf("PromptFor(a/b) = %q", got)
	}
	if got := c.PromptFor("a", "c"); got != "global" {
		t.Errorf("PromptFor(a/c) = %q", got)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	s := NewStoreAt(t.TempDir())
	if c, err := s.Load(); err != nil || c.PollIntervalMinutes != DefaultPollMinutes {
		t.Fatalf("Load vazio = %+v, %v", c, err)
	}
	if _, err := s.Save(Config{Repos: []RepoConfig{{Owner: "a", Name: "b"}}}); err != nil {
		t.Fatal(err)
	}
	c, _ := s.Load()
	if len(c.Repos) != 1 || c.Repos[0].FullName() != "a/b" {
		t.Errorf("Load = %+v", c)
	}
}

func TestLoadDefaultsLaunchAtLoginOnFirstRun(t *testing.T) {
	s := NewStoreAt(t.TempDir())
	c, err := s.Load()
	if err != nil || !c.LaunchAtLogin {
		t.Fatalf("primeira execução: LaunchAtLogin = %v, err = %v; esperado ligado", c.LaunchAtLogin, err)
	}
	c.LaunchAtLogin = false
	if _, err := s.Save(c); err != nil {
		t.Fatal(err)
	}
	if c, _ := s.Load(); c.LaunchAtLogin {
		t.Errorf("depois de desligar e salvar, LaunchAtLogin deveria continuar desligado")
	}
}

func TestNormalizeFallsBackToCodeMode(t *testing.T) {
	if got := (Config{DefaultMode: "qualquer"}).Normalize().DefaultMode; got != ModeCode {
		t.Errorf("DefaultMode inválido = %q", got)
	}
	if got := (Config{DefaultMode: ModeArchitect}).Normalize().DefaultMode; got != ModeArchitect {
		t.Errorf("DefaultMode architect = %q", got)
	}
}

func TestNormalizeEffort(t *testing.T) {
	cases := map[string]string{"": "", " High ": "high", "xhigh": "xhigh", "turbo": ""}
	for in, want := range cases {
		if got := (Config{Effort: in}).Normalize().Effort; got != want {
			t.Errorf("Effort %q: got %q, want %q", in, got, want)
		}
	}
}
