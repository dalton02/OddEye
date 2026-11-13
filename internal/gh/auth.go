package gh

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"oddeye/internal/proc"
)

// RequiredScopes são os escopos que o app usa: repo para ler PRs privadas e publicar reviews,
// read:org para listar os repositórios das organizações do usuário.
var RequiredScopes = []string{"repo", "read:org"}

var scopesLine = regexp.MustCompile(`Token scopes:\s*(.*)`)

// parseMissingScopes lê a saída do `gh auth status --active` e devolve os escopos obrigatórios
// ausentes. Sem a linha de escopos (GH_TOKEN, token fine-grained) não há como saber: devolve nil
// para não pedir autorização à toa.
func parseMissingScopes(status string) []string {
	m := scopesLine.FindStringSubmatch(status)
	if m == nil {
		return nil
	}
	have := map[string]bool{}
	for _, s := range strings.Split(m[1], ",") {
		have[strings.Trim(strings.TrimSpace(s), "'\"")] = true
	}
	var missing []string
	for _, s := range RequiredScopes {
		if !have[s] {
			missing = append(missing, s)
		}
	}
	return missing
}

// MissingScopes devolve os escopos obrigatórios que o token ativo do gh não tem.
func (r Runner) MissingScopes(ctx context.Context) []string {
	cmd := exec.CommandContext(ctx, "gh", "auth", "status", "--active", "--hostname", "github.com")
	proc.HideWindow(cmd)
	out, _ := cmd.CombinedOutput()
	return parseMissingScopes(string(out))
}

var (
	deviceCode = regexp.MustCompile(`\b([A-Z0-9]{4}-[A-Z0-9]{4})\b`)
	deviceURL  = regexp.MustCompile(`https://\S+/login/device\S*`)
)

const defaultDeviceURL = "https://github.com/login/device"

// DeviceCode é o código de uso único que o usuário digita no GitHub para autorizar o gh.
type DeviceCode struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}

// Authorize faz o device flow do gh sem terminal: faz login se não houver conta, senão amplia os
// escopos da conta ativa. onCode recebe o código assim que o gh o imprime; a função retorna quando o
// usuário conclui (ou cancela, via ctx) a autorização no navegador.
func (r Runner) Authorize(ctx context.Context, loggedIn bool, onCode func(DeviceCode)) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return errors.New("GitHub CLI (gh) não encontrado no PATH: instale em https://cli.github.com")
	}
	scopes := strings.Join(RequiredScopes, ",")
	args := []string{"auth", "refresh", "--hostname", "github.com", "--scopes", scopes}
	if !loggedIn {
		args = []string{"auth", "login", "--web", "--hostname", "github.com", "--git-protocol", "https", "--scopes", scopes}
	}
	cmd := exec.CommandContext(ctx, "gh", args...)
	proc.HideWindow(cmd)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("não consegui iniciar o gh: %w", err)
	}
	var transcript strings.Builder
	done := make(chan struct{})
	go func() {
		defer close(done)
		sent := false
		sc := bufio.NewScanner(pr)
		for sc.Scan() {
			line := sc.Text()
			transcript.WriteString(line + "\n")
			if code := deviceCode.FindString(line); code != "" && !sent {
				sent = true
				dc := DeviceCode{Code: code, URL: defaultDeviceURL}
				if u := deviceURL.FindString(transcript.String()); u != "" {
					dc.URL = u
				}
				onCode(dc)
				// Num terminal o gh espera um Enter antes de abrir o navegador; sem TTY ele não pede,
				// e o Enter extra é ignorado.
				_, _ = io.WriteString(stdin, "\n")
			}
		}
	}()
	err = cmd.Wait()
	_ = pw.Close()
	<-done
	if ctx.Err() != nil {
		return errors.New("autorização cancelada")
	}
	if err != nil {
		msg := strings.TrimSpace(transcript.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("gh auth: %s", msg)
	}
	return nil
}
