// Package claude roda o Claude Code em modo headless (`claude -p`) para triar os arquivos
// críticos de uma PR e explicar cada um, transmitindo o progresso enquanto ele pensa.
package claude

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"oddeye/internal/gh"
	"oddeye/internal/proc"
	"oddeye/internal/review"
)

type Attention struct {
	Line int    `json:"line"`
	Note string `json:"note"`
}

// Reference é um arquivo citado na explicação que não está entre os escolhidos. Path vem como o
// Claude o viu no diff (caminho completo ou de import); quem exibe resolve para o arquivo real.
type Reference struct {
	Path string `json:"path"`
	Why  string `json:"why"`
}

type FileAnalysis struct {
	Path        string      `json:"path"`
	Severity    string      `json:"severity"` // critical, high, medium
	Why         string      `json:"why"`
	Explanation string      `json:"explanation"`
	Attention   []Attention `json:"attention"`
	References  []Reference `json:"references"`
}

type Analysis struct {
	Summary string         `json:"summary"`
	Files   []FileAnalysis `json:"files"`
}

type Progress struct {
	Stage   string `json:"stage"` // starting, thinking, writing, done
	Message string `json:"message"`
	Files   int    `json:"files"`
}

const schema = `{
  "type": "object",
  "properties": {
    "summary": {"type": "string"},
    "files": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "path": {"type": "string"},
          "severity": {"type": "string", "enum": ["critical", "high", "medium"]},
          "why": {"type": "string"},
          "explanation": {"type": "string"},
          "attention": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {"line": {"type": "integer"}, "note": {"type": "string"}},
              "required": ["line", "note"]
            }
          },
          "references": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {"path": {"type": "string"}, "why": {"type": "string"}},
              "required": ["path", "why"]
            }
          }
        },
        "required": ["path", "severity", "why", "explanation", "attention", "references"]
      }
    }
  },
  "required": ["summary", "files"]
}`

// Mode é o tipo de leitura que o revisor faz da PR.
type Mode string

const (
	ModeCode      Mode = "code"      // triagem dos arquivos críticos, com o código na tela
	ModeArchitect Mode = "architect" // abordagem, trade-offs e arquitetura, sem falar de código
)

// ParseMode aceita o valor vindo da configuração ou do frontend; o que não reconhece vira ModeCode.
func ParseMode(s string) Mode {
	if Mode(strings.TrimSpace(s)) == ModeArchitect {
		return ModeArchitect
	}
	return ModeCode
}

const (
	maxPatchLines = 400
	maxPromptSize = 300_000
	MaxFiles      = 15
)

// ResolveModel aplica a precedência do modelo: a variável ODDEYE_MODEL sobrescreve a configuração.
func ResolveModel(configured string) string {
	if m := os.Getenv("ODDEYE_MODEL"); m != "" {
		return m
	}
	return strings.TrimSpace(configured)
}

func Available() error {
	if _, err := exec.LookPath("claude"); err != nil {
		return errors.New("Claude Code (claude) não encontrado no PATH: instale em https://claude.com/claude-code e faça login")
	}
	return nil
}

// DefaultReviewerPrompt é a parte editável do pedido: quem é o revisor, o que conta como crítico,
// o tom e o idioma. O usuário pode trocá-la nas configurações (global ou por repositório). Sem quebra de
// linha no meio das frases, porque o texto aparece num editor que já quebra sozinho.
const DefaultReviewerPrompt = `Você é um revisor de código sênior. Sua tarefa é fazer a TRIAGEM de uma Pull Request: escolher somente os arquivos que um revisor humano PRECISA ler com atenção (regra de negócio, dinheiro, segurança, concorrência, contratos públicos, schema de banco, migrations, mudanças de comportamento), e explicar cada um.

- Ignore testes e documentação, mesmo que algum tenha escapado do filtro.
- Ignore mudanças triviais (renomeação, formatação, imports, tipos sem efeito, constantes óbvias).
- "why": uma frase dizendo por que o arquivo merece atenção.
- "explanation": 2 a 5 frases, em tom de conversa com o revisor, explicando o que o arquivo faz e o que mudou nele.
- "attention": de 0 a 4 pontos concretos que o revisor deve conferir (possíveis bugs, corridas, casos de borda). Não invente problema: se não houver, deixe vazio.
- "summary": 2 a 3 frases com a visão geral da PR.
- Escreva tudo em português do Brasil.`

// formatContract é a parte fixa do pedido: as regras de que o parse, o Sanitize e os links de arquivo
// dependem. Fica fora do editor para que um prompt customizado não quebre a leitura da resposta.
const formatContract = `# Formato da resposta (obrigatório, tem prioridade sobre qualquer instrução acima)
- Escolha no máximo %d arquivos, ordenados do mais crítico para o menos crítico.
- "path" deve ser exatamente um dos caminhos listados em "Arquivos candidatos".
- "severity" é critical, high ou medium.
- Cada ponto de "attention" leva o número da linha no ARQUIVO NOVO. Os números entre colchetes no início
  das linhas do diff já são essas linhas.
- Sempre que citar um arquivo (no summary, na explanation ou no attention), escreva o caminho entre crases
  exatamente como ele aparece no diff: o caminho completo da lista de arquivos (ex.: ` + "`src/a/b.ts`" + `) ou o caminho
  do import (ex.: ` + "`@infra/repositories/prisma/user`" + `). Nunca cite um arquivo só pelo nome (` + "`index.ts`" + `), e nunca
  invente caminho que não aparece no diff.
- "references": todo arquivo que você citou na explanation ou no attention daquele arquivo e que NÃO está
  entre os arquivos escolhidos, com o mesmo caminho que você escreveu e uma frase dizendo por que ele importa.
  Deixe vazio se não citou nenhum.
`

// Instructions devolve as instruções efetivas: as do usuário ou, se vazias, as padrão.
func Instructions(custom string) string {
	if strings.TrimSpace(custom) == "" {
		return DefaultReviewerPrompt
	}
	return strings.TrimSpace(custom)
}

// CacheKey resume o que muda o resultado da análise além do commit (instruções e modelo), para que
// trocar o prompt ou o modelo não devolva a análise antiga do cache.
func CacheKey(instructions, model string) string {
	return cacheKey(Instructions(instructions), model)
}

func cacheKey(effective, model string) string {
	sum := sha256.Sum256([]byte(effective + "\x00" + model))
	return hex.EncodeToString(sum[:])[:12]
}

// BuildPrompt monta o pedido de triagem com os patches numerados pelo lado novo do arquivo,
// para que as linhas de atenção que o Claude devolver batam com o arquivo exibido. instructions vazio
// usa DefaultReviewerPrompt; o contrato de formato entra sempre.
func BuildPrompt(pr gh.PRInfo, candidates []gh.File, skipped []review.Skipped, instructions string) string {
	var b strings.Builder
	b.WriteString(Instructions(instructions))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, formatContract, MaxFiles)
	writePR(&b, pr, skipped, "## Arquivos já descartados (não escolha estes)")
	writeCandidates(&b, candidates)
	return b.String()
}

// writePR escreve o cabeçalho da PR e a lista dos arquivos que ficaram fora dos candidatos.
func writePR(b *strings.Builder, pr gh.PRInfo, skipped []review.Skipped, skippedTitle string) {
	fmt.Fprintf(b, `
# PR: %s
Autor: %s | %s -> %s | +%d -%d

## Descrição
%s

`, pr.Title, pr.Author.Login, pr.HeadRefName, pr.BaseRefName, pr.Additions, pr.Deletions, strings.TrimSpace(pr.Body))

	if len(skipped) > 0 {
		b.WriteString(skippedTitle + "\n")
		for _, s := range skipped {
			fmt.Fprintf(b, "- %s (%s)\n", s.Path, s.Reason)
		}
		b.WriteString("\n")
	}
}

// writeCandidates escreve os diffs numerados até o orçamento do prompt; o que não cabe entra sem diff.
func writeCandidates(b *strings.Builder, candidates []gh.File) {
	b.WriteString("## Arquivos candidatos\n\n")
	budget := maxPromptSize - b.Len()
	for _, f := range candidates {
		header := fmt.Sprintf("### %s (%s, +%d -%d)\n", f.Path, f.Status, f.Additions, f.Deletions)
		body := numberPatch(f.Patch)
		if len(header)+len(body) > budget {
			body = "(diff omitido por tamanho)\n"
		}
		block := header + "```diff\n" + body + "```\n\n"
		budget -= len(block)
		b.WriteString(block)
	}
}

var hunkHeader = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func numberPatch(patch string) string {
	if patch == "" {
		return "(sem diff textual)\n"
	}
	var b strings.Builder
	lines := strings.Split(patch, "\n")
	right := 0
	for i, l := range lines {
		if i >= maxPatchLines {
			fmt.Fprintf(&b, "... (%d linhas restantes omitidas)\n", len(lines)-i)
			break
		}
		if m := hunkHeader.FindStringSubmatch(l); m != nil {
			fmt.Sscanf(m[1], "%d", &right)
			b.WriteString(l + "\n")
			continue
		}
		switch {
		case strings.HasPrefix(l, "-"), strings.HasPrefix(l, `\`):
			b.WriteString("       " + l + "\n")
		default:
			fmt.Fprintf(&b, "[%4d] %s\n", right, l)
			right++
		}
	}
	return b.String()
}

type streamEvent struct {
	Type             string          `json:"type"`
	Subtype          string          `json:"subtype"`
	IsError          bool            `json:"is_error"`
	Result           string          `json:"result"`
	StructuredOutput json.RawMessage `json:"structured_output"`
	Event            *struct {
		Type  string `json:"type"`
		Delta *struct {
			Type        string `json:"type"`
			Thinking    string `json:"thinking"`
			Text        string `json:"text"`
			PartialJSON string `json:"partial_json"`
		} `json:"delta"`
	} `json:"event"`
}

var pathInJSON = regexp.MustCompile(`"path"\s*:\s*"([^"]+)"`)

// Analyze roda o Claude e devolve a triagem. onProgress recebe atualizações do stream.
func Analyze(ctx context.Context, prompt, model string, onProgress func(Progress)) (Analysis, error) {
	var a Analysis
	describe := func(partial string) (string, int) {
		matches := pathInJSON.FindAllStringSubmatch(partial, -1)
		if len(matches) == 0 {
			return "", 0
		}
		return "Explicando " + matches[len(matches)-1][1], len(matches)
	}
	if err := run(ctx, prompt, model, schema, &a, onProgress, describe); err != nil {
		return Analysis{}, err
	}
	onProgress(Progress{Stage: "done", Message: "Pronto!", Files: len(a.Files)})
	return a, nil
}

// run executa o Claude Code headless com o schema dado e decodifica a resposta em out. describe lê o
// JSON parcial que o Claude já escreveu e devolve a mensagem de progresso e quantos itens estão prontos;
// só vira evento quando a contagem avança.
func run(ctx context.Context, prompt, model, schema string, out any, onProgress func(Progress), describe func(partial string) (string, int)) error {
	args := []string{
		"-p",
		"--output-format", "stream-json",
		"--verbose",
		"--include-partial-messages",
		"--tools", "",
		"--strict-mcp-config",
		"--no-session-persistence",
		"--json-schema", schema,
	}
	if m := ResolveModel(model); m != "" {
		args = append(args, "--model", m)
	}
	cmd := exec.CommandContext(ctx, "claude", args...)
	proc.HideWindow(cmd)
	// Fora do repositório do usuário: o prompt é autossuficiente e não queremos o CLAUDE.md de projeto.
	cmd.Dir = os.TempDir()
	cmd.Stdin = strings.NewReader(prompt)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("não consegui iniciar o claude: %w", err)
	}
	onProgress(Progress{Stage: "starting", Message: "Abrindo o diff e lendo a descrição da PR..."})

	var (
		final    *streamEvent
		thinking strings.Builder
		partial  strings.Builder
		seen     int
	)
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 1<<20), 64<<20)
	for scanner.Scan() {
		var ev streamEvent
		if json.Unmarshal(scanner.Bytes(), &ev) != nil {
			continue
		}
		switch ev.Type {
		case "result":
			e := ev
			final = &e
		case "stream_event":
			if ev.Event == nil || ev.Event.Delta == nil {
				continue
			}
			d := ev.Event.Delta
			switch d.Type {
			case "thinking_delta":
				thinking.WriteString(d.Thinking)
				onProgress(Progress{Stage: "thinking", Message: tail(thinking.String(), 220), Files: seen})
			case "input_json_delta":
				partial.WriteString(d.PartialJSON)
				if msg, n := describe(partial.String()); n > seen {
					seen = n
					onProgress(Progress{Stage: "writing", Message: msg, Files: seen})
				}
			}
		}
	}
	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return errors.New("análise cancelada")
	}
	if final == nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" && waitErr != nil {
			msg = waitErr.Error()
		}
		return fmt.Errorf("o claude terminou sem resultado: %s", msg)
	}
	if final.IsError {
		return fmt.Errorf("o claude retornou erro (%s): %s%s", final.Subtype, final.Result, loginHint(final.Result))
	}
	raw := final.StructuredOutput
	if len(raw) == 0 || string(raw) == "null" {
		raw = json.RawMessage(extractJSON(final.Result))
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("resposta do claude fora do formato esperado: %w", err)
	}
	return nil
}

// loginHint explica como resolver quando o Claude Code está instalado mas sem login: o app não tem como
// fazer esse login por conta própria, ele acontece no terminal do usuário.
func loginHint(result string) string {
	r := strings.ToLower(result)
	if strings.Contains(r, "login") || strings.Contains(r, "auth") || strings.Contains(r, "api key") {
		return "\n\nAbra o terminal, rode `claude` e faça login (/login). Depois tente de novo."
	}
	return ""
}

func tail(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return "…" + string(r[len(r)-n:])
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "{"); i >= 0 {
		if j := strings.LastIndex(s, "}"); j > i {
			return s[i : j+1]
		}
	}
	return s
}

// Sanitize descarta caminhos que não estão entre os candidatos e corta no limite.
func Sanitize(a Analysis, candidates []gh.File) Analysis {
	valid := map[string]bool{}
	for _, f := range candidates {
		valid[f.Path] = true
	}
	out := Analysis{Summary: a.Summary}
	used := map[string]bool{}
	for _, f := range a.Files {
		if !valid[f.Path] || used[f.Path] {
			continue
		}
		used[f.Path] = true
		if f.Attention == nil {
			f.Attention = []Attention{}
		}
		if f.References == nil {
			f.References = []Reference{}
		}
		out.Files = append(out.Files, f)
		if len(out.Files) == MaxFiles {
			break
		}
	}
	return out
}
