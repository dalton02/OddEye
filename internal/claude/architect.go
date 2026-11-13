package claude

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"oddeye/internal/gh"
	"oddeye/internal/review"
)

// Blueprint é a leitura de arquitetura de uma PR: o que mudou no sistema, como as peças conversam,
// que decisões foram tomadas e o que elas custam. Nenhum texto dele cita função ou arquivo: os
// arquivos de cada componente ficam só em Node.Files, para o app medir o peso da mudança e abrir o código.
type Blueprint struct {
	Headline  string       `json:"headline"`
	Kind      string       `json:"kind"`
	Summary   string       `json:"summary"`
	Before    string       `json:"before"`
	After     string       `json:"after"`
	Nodes     []Node       `json:"nodes"`
	Edges     []Edge       `json:"edges"`
	Flows     []Flow       `json:"flows"`
	Decisions []Decision   `json:"decisions"`
	Impact    []ImpactAxis `json:"impact"`
	Risks     []Risk       `json:"risks"`
	Questions []string     `json:"questions"`
}

// Weight é o tamanho real da mudança num componente, somado dos arquivos dele na PR.
type Weight struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
}

type Node struct {
	ID     string   `json:"id"`
	Label  string   `json:"label"`
	Kind   string   `json:"kind"`   // actor, interface, service, worker, store, queue, external
	Change string   `json:"change"` // new, changed, removed, context
	Role   string   `json:"role"`
	Files  []string `json:"files"`
	Weight Weight   `json:"weight"` // calculado no SanitizeBlueprint, nunca pelo Claude
}

type Edge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Label  string `json:"label"`
	Style  string `json:"style"`  // sync, async, data
	Change string `json:"change"` // new, changed, removed, context
}

type Step struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Action string `json:"action"`
	Note   string `json:"note"`
	Change string `json:"change"` // new, changed, context
}

type Flow struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Steps   []Step `json:"steps"`
}

type Alternative struct {
	Name   string `json:"name"`
	WhyNot string `json:"whyNot"`
}

type Decision struct {
	Title        string        `json:"title"`
	Chosen       string        `json:"chosen"`
	Why          string        `json:"why"`
	Gains        []string      `json:"gains"`
	Costs        []string      `json:"costs"`
	Alternatives []Alternative `json:"alternatives"`
}

type ImpactAxis struct {
	Axis  string `json:"axis"`
	Score int    `json:"score"` // -2 piora muito ... +2 melhora muito
	Note  string `json:"note"`
}

type Risk struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Level  string `json:"level"` // high, medium, low
	Node   string `json:"node"`  // componente onde o risco mora; vazio se for da PR inteira
}

const (
	MaxNodes     = 12
	MaxEdges     = 16
	MaxFlows     = 3
	MaxSteps     = 8
	MaxDecisions = 4
	MaxRisks     = 6
	MaxQuestions = 6

	maxTreeLines = 160
)

// ImpactAxes são os eixos fixos do gráfico de impacto, na ordem em que aparecem. Todos têm o mesmo
// sentido (positivo = o sistema fica melhor nesse aspecto), para o gráfico ser lido de relance.
var ImpactAxes = []string{"simplicity", "reliability", "performance", "maintainability", "decoupling", "operability"}

var (
	nodeKinds   = []string{"actor", "interface", "service", "worker", "store", "queue", "external"}
	changeKinds = []string{"new", "changed", "removed", "context"}
	stepChanges = []string{"new", "changed", "context"}
	edgeStyles  = []string{"sync", "async", "data"}
	riskLevels  = []string{"high", "medium", "low"}
	prKinds     = []string{"feature", "fix", "refactor", "performance", "security", "infra", "chore"}
)

func enum(values []string) string {
	return `"` + strings.Join(values, `", "`) + `"`
}

var architectSchema = `{
  "type": "object",
  "properties": {
    "headline": {"type": "string"},
    "kind": {"type": "string", "enum": [` + enum(prKinds) + `]},
    "summary": {"type": "string"},
    "before": {"type": "string"},
    "after": {"type": "string"},
    "nodes": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "id": {"type": "string"},
          "label": {"type": "string"},
          "kind": {"type": "string", "enum": [` + enum(nodeKinds) + `]},
          "change": {"type": "string", "enum": [` + enum(changeKinds) + `]},
          "role": {"type": "string"},
          "files": {"type": "array", "items": {"type": "string"}}
        },
        "required": ["id", "label", "kind", "change", "role", "files"]
      }
    },
    "edges": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "from": {"type": "string"},
          "to": {"type": "string"},
          "label": {"type": "string"},
          "style": {"type": "string", "enum": [` + enum(edgeStyles) + `]},
          "change": {"type": "string", "enum": [` + enum(changeKinds) + `]}
        },
        "required": ["from", "to", "label", "style", "change"]
      }
    },
    "flows": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "title": {"type": "string"},
          "summary": {"type": "string"},
          "steps": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "from": {"type": "string"},
                "to": {"type": "string"},
                "action": {"type": "string"},
                "note": {"type": "string"},
                "change": {"type": "string", "enum": [` + enum(stepChanges) + `]}
              },
              "required": ["from", "to", "action", "note", "change"]
            }
          }
        },
        "required": ["title", "summary", "steps"]
      }
    },
    "decisions": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "title": {"type": "string"},
          "chosen": {"type": "string"},
          "why": {"type": "string"},
          "gains": {"type": "array", "items": {"type": "string"}},
          "costs": {"type": "array", "items": {"type": "string"}},
          "alternatives": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {"name": {"type": "string"}, "whyNot": {"type": "string"}},
              "required": ["name", "whyNot"]
            }
          }
        },
        "required": ["title", "chosen", "why", "gains", "costs", "alternatives"]
      }
    },
    "impact": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "axis": {"type": "string", "enum": [` + enum(ImpactAxes) + `]},
          "score": {"type": "integer"},
          "note": {"type": "string"}
        },
        "required": ["axis", "score", "note"]
      }
    },
    "risks": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "title": {"type": "string"},
          "detail": {"type": "string"},
          "level": {"type": "string", "enum": [` + enum(riskLevels) + `]},
          "node": {"type": "string"}
        },
        "required": ["title", "detail", "level", "node"]
      }
    },
    "questions": {"type": "array", "items": {"type": "string"}}
  },
  "required": ["headline", "kind", "summary", "before", "after", "nodes", "edges", "flows", "decisions", "impact", "risks", "questions"]
}`

// DefaultArchitectPrompt é a parte editável do modo Arquiteto: quem ele é, o que observa e o tom.
// Sem quebra de linha no meio das frases, porque o texto aparece num editor que já quebra sozinho.
const DefaultArchitectPrompt = `Você é um arquiteto de software explicando uma Pull Request para um colega dev que ainda não abriu o diff. Seu foco NÃO é o código: é a abordagem escolhida, como as peças do sistema se encaixam, os trade-offs e o que muda na arquitetura.

- Escreva como numa conversa de corredor entre devs: direto, concreto, semi-técnico. Pode usar termos de arquitetura (fila, cache, idempotência, consistência), mas explique o efeito prático de cada um.
- Conte primeiro o problema que a PR resolve e depois a solução. O leitor precisa sair entendendo o que o sistema passa a fazer e por quê.
- Trade-offs são o coração da análise: toda decisão relevante tem um ganho e um custo. Diga os dois com honestidade e cite as alternativas que seriam razoáveis.
- Não elogie nem critique por educação. Se a abordagem é boa, diga por quê; se cria uma dívida, diga qual.
- Não invente: se o diff não mostra uma parte do sistema, não desenhe essa parte.
- Escreva tudo em português do Brasil.`

// architectContract é a parte fixa do pedido: o que cada campo significa e as regras de que o
// SanitizeBlueprint e os diagramas dependem. Fica fora do editor para um prompt customizado não quebrar a tela.
const architectContract = `# Formato da resposta (obrigatório, tem prioridade sobre qualquer instrução acima)

## Regra de ouro: nada de código no texto
- Nunca cite nome de função, método, classe, variável, arquivo, pasta, tabela ou coluna em nenhum campo de texto.
  Descreva pelo papel: "o serviço que aprova saques", "a tabela de saldos", "a rotina que roda de madrugada".
- Nunca use crases, trechos de código, assinaturas ou números de linha.
- A única exceção é "files" de cada componente, que é uma lista de caminhos e não aparece como texto.

## Visão geral
- "headline": uma frase curta (até 90 caracteres) dizendo o que a PR faz, como uma manchete.
- "kind": o tipo da mudança.
- "summary": 2 a 3 frases com o problema e a solução.
- "before": 1 a 2 frases de como o sistema se comportava antes. "after": 1 a 2 frases de como passa a se comportar.

## Mapa do sistema ("nodes" e "edges")
- "nodes": de 3 a %d componentes do sistema que a PR toca ou dos quais ela depende diretamente. Um componente
  é uma peça com responsabilidade própria (um serviço, uma tela, uma fila, um banco, um sistema externo), não um arquivo.
- "id": identificador curto em kebab-case, único. "label": nome legível de até 3 palavras.
- "kind": actor (pessoa ou sistema que inicia a ação), interface (tela, API, CLI), service (regra de negócio),
  worker (processamento em segundo plano), store (banco, cache, arquivo), queue (fila, tópico, evento), external (serviço de terceiros).
- "change": new (criado pela PR), changed (existia e mudou), removed (a PR elimina), context (não muda, mas
  ajuda a entender o desenho).
- "role": uma frase dizendo o que esse componente faz no sistema e o que muda nele com a PR.
- "files": os caminhos, exatamente como aparecem em "Arquivos candidatos", que pertencem ao componente.
  Cada arquivo em no máximo um componente. Vazio para componentes de contexto.
- "edges": até %d ligações entre componentes. "from" e "to" são ids de "nodes". "label": verbo curto
  (até 4 palavras) do que passa por ali. "style": sync (chamada que espera resposta), async (evento, fila,
  segundo plano), data (leitura ou escrita de dados). "change" como nos componentes.

## Fluxos ("flows")
- De 1 a %d fluxos, cada um contando do início ao fim um caminho importante que a PR cria ou altera.
- "title": nome do fluxo. "summary": uma frase dizendo quando ele acontece.
- "steps": de 2 a %d passos em ordem. "from" e "to" são ids de "nodes" (iguais quando o passo é interno ao
  componente). "action": o que acontece, em até 8 palavras. "note": uma frase explicando por que esse passo
  existe ou o que ele garante. "change": new, changed ou context.

## Decisões ("decisions")
- Até %d decisões de desenho que a PR toma. "title": a pergunta que a decisão responde.
- "chosen": o caminho escolhido. "why": por que ele faz sentido aqui.
- "gains" e "costs": de 1 a 3 itens curtos cada. Toda decisão tem custo: não deixe "costs" vazio.
- "alternatives": de 0 a 2 caminhos razoáveis que não foram seguidos, com "whyNot".

## Impacto ("impact")
- Um item para cada eixo: simplicity (facilidade de entender), reliability (resistência a falhas e a dados
  errados), performance (velocidade e uso de recursos), maintainability (facilidade de mudar depois),
  decoupling (independência entre as peças), operability (facilidade de rodar, observar e reverter).
- "score": inteiro de -2 a 2. Positivo significa que o sistema MELHORA nesse eixo, negativo que PIORA,
  0 que não muda. Use 0 sem medo: a maioria das PRs não mexe em todos os eixos.
- "note": uma frase justificando a nota.

## Riscos e perguntas
- "risks": até %d riscos de arquitetura (não bugs de linha): acoplamento, ponto único de falha, migração,
  consistência, escala, reversão. "node" é o id do componente onde o risco mora, ou vazio. Não invente risco.
- "questions": até %d perguntas que valeria fazer ao autor antes de aprovar, escritas prontas para colar
  num comentário de review.
`

// ArchitectInstructions devolve as instruções efetivas do modo Arquiteto: as do usuário ou as padrão.
func ArchitectInstructions(custom string) string {
	if strings.TrimSpace(custom) == "" {
		return DefaultArchitectPrompt
	}
	return strings.TrimSpace(custom)
}

// ArchitectCacheKey é a CacheKey do modo Arquiteto.
func ArchitectCacheKey(instructions, model string) string {
	return cacheKey(string(ModeArchitect)+"\x00"+ArchitectInstructions(instructions), model)
}

// BuildArchitectPrompt monta o pedido do modo Arquiteto. tree é a lista de arquivos do repositório no
// head da PR (pode ser vazia): entra resumida, para o Claude situar a mudança no sistema.
func BuildArchitectPrompt(pr gh.PRInfo, candidates []gh.File, skipped []review.Skipped, tree []string, instructions string) string {
	var b strings.Builder
	b.WriteString(ArchitectInstructions(instructions))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, architectContract, MaxNodes, MaxEdges, MaxFlows, MaxSteps, MaxDecisions, MaxRisks, MaxQuestions)
	writePR(&b, pr, skipped, "## Arquivos fora da análise (testes, docs, gerados: servem só de pista do que a PR cobre)")
	if summary := treeSummary(tree); summary != "" {
		b.WriteString("## Estrutura do repositório (pastas e quantidade de arquivos)\n")
		b.WriteString(summary)
		b.WriteString("\n")
	}
	writeCandidates(&b, candidates)
	return b.String()
}

// treeSummary resume a árvore em pastas com a contagem de arquivos, na maior profundidade (até 3
// níveis) que ainda cabe em maxTreeLines.
func treeSummary(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	for depth := 3; depth >= 1; depth-- {
		counts := map[string]int{}
		for _, p := range paths {
			parts := strings.Split(p, "/")
			dir := "(raiz)"
			if len(parts) > 1 {
				dir = strings.Join(parts[:min(depth, len(parts)-1)], "/") + "/"
			}
			counts[dir]++
		}
		if len(counts) > maxTreeLines && depth > 1 {
			continue
		}
		dirs := make([]string, 0, len(counts))
		for d := range counts {
			dirs = append(dirs, d)
		}
		sort.Strings(dirs)
		var b strings.Builder
		for i, d := range dirs {
			if i == maxTreeLines {
				fmt.Fprintf(&b, "... (%d pastas omitidas)\n", len(dirs)-i)
				break
			}
			fmt.Fprintf(&b, "- %s (%d)\n", d, counts[d])
		}
		return b.String()
	}
	return ""
}

// architectStages são as chaves do JSON na ordem em que o Claude as escreve, com a mensagem de progresso.
var architectStages = []struct{ key, message string }{
	{`"nodes"`, "Desenhando o mapa do sistema..."},
	{`"flows"`, "Traçando os fluxos passo a passo..."},
	{`"decisions"`, "Pesando as decisões e os trade-offs..."},
	{`"impact"`, "Medindo o impacto na arquitetura..."},
	{`"risks"`, "Procurando riscos e perguntas para o autor..."},
}

// AnalyzeArchitecture roda o Claude no modo Arquiteto. O resultado ainda precisa do SanitizeBlueprint.
func AnalyzeArchitecture(ctx context.Context, prompt, model, effort string, onProgress func(Progress)) (Blueprint, error) {
	var bp Blueprint
	describe := func(partial string) (string, int) {
		msg, n := "", 0
		for _, s := range architectStages {
			if !strings.Contains(partial, s.key) {
				break
			}
			msg, n = s.message, n+1
		}
		return msg, n
	}
	if err := run(ctx, prompt, model, effort, architectSchema, &bp, onProgress, describe); err != nil {
		return Blueprint{}, err
	}
	onProgress(Progress{Stage: "done", Message: "Pronto!"})
	return bp, nil
}

// SanitizeBlueprint deixa o blueprint seguro para desenhar: sem referência a componente que não existe,
// sem arquivo de fora da PR, dentro dos limites e sem crases no texto. Também calcula o peso real da
// mudança em cada componente a partir dos arquivos da PR.
func SanitizeBlueprint(bp Blueprint, files []gh.File) Blueprint {
	byPath := map[string]gh.File{}
	for _, f := range files {
		byPath[f.Path] = f
	}
	out := Blueprint{
		Headline: plain(bp.Headline), Kind: oneOf(bp.Kind, prKinds, "feature"), Summary: plain(bp.Summary),
		Before: plain(bp.Before), After: plain(bp.After),
		Nodes: []Node{}, Edges: []Edge{}, Flows: []Flow{}, Decisions: []Decision{}, Impact: []ImpactAxis{},
		Risks: []Risk{}, Questions: []string{},
	}

	nodes := map[string]bool{}
	claimed := map[string]bool{}
	for _, n := range bp.Nodes {
		n.ID = strings.TrimSpace(n.ID)
		if n.ID == "" || nodes[n.ID] || len(out.Nodes) == MaxNodes {
			continue
		}
		nodes[n.ID] = true
		n.Label, n.Role = plain(n.Label), plain(n.Role)
		if n.Label == "" {
			n.Label = n.ID
		}
		n.Kind, n.Change = oneOf(n.Kind, nodeKinds, "service"), oneOf(n.Change, changeKinds, "context")
		kept := []string{}
		n.Weight = Weight{}
		for _, p := range n.Files {
			f, ok := byPath[p]
			if !ok || claimed[p] {
				continue
			}
			claimed[p] = true
			kept = append(kept, p)
			n.Weight.Additions += f.Additions
			n.Weight.Deletions += f.Deletions
		}
		n.Files = kept
		out.Nodes = append(out.Nodes, n)
	}

	linked := map[string]bool{}
	for _, e := range bp.Edges {
		key := e.From + "\x00" + e.To
		if !nodes[e.From] || !nodes[e.To] || e.From == e.To || linked[key] || len(out.Edges) == MaxEdges {
			continue
		}
		linked[key] = true
		e.Label = plain(e.Label)
		e.Style, e.Change = oneOf(e.Style, edgeStyles, "sync"), oneOf(e.Change, changeKinds, "context")
		out.Edges = append(out.Edges, e)
	}

	for _, f := range bp.Flows {
		flow := Flow{Title: plain(f.Title), Summary: plain(f.Summary), Steps: []Step{}}
		for _, s := range f.Steps {
			if !nodes[s.From] || !nodes[s.To] || len(flow.Steps) == MaxSteps {
				continue
			}
			s.Action, s.Note = plain(s.Action), plain(s.Note)
			s.Change = oneOf(s.Change, stepChanges, "context")
			flow.Steps = append(flow.Steps, s)
		}
		if len(flow.Steps) > 0 && len(out.Flows) < MaxFlows {
			out.Flows = append(out.Flows, flow)
		}
	}

	for _, d := range bp.Decisions {
		if len(out.Decisions) == MaxDecisions {
			break
		}
		d.Title, d.Chosen, d.Why = plain(d.Title), plain(d.Chosen), plain(d.Why)
		d.Gains, d.Costs = plainList(d.Gains, 3), plainList(d.Costs, 3)
		alts := []Alternative{}
		for _, a := range d.Alternatives {
			if a.Name = plain(a.Name); a.Name != "" && len(alts) < 2 {
				alts = append(alts, Alternative{Name: a.Name, WhyNot: plain(a.WhyNot)})
			}
		}
		d.Alternatives = alts
		if d.Title != "" || d.Chosen != "" {
			out.Decisions = append(out.Decisions, d)
		}
	}

	scores := map[string]ImpactAxis{}
	for _, i := range bp.Impact {
		if _, dup := scores[i.Axis]; !dup {
			scores[i.Axis] = i
		}
	}
	for _, axis := range ImpactAxes {
		i, ok := scores[axis]
		if !ok {
			continue
		}
		out.Impact = append(out.Impact, ImpactAxis{Axis: axis, Score: max(-2, min(2, i.Score)), Note: plain(i.Note)})
	}

	for _, r := range bp.Risks {
		r.Title, r.Detail = plain(r.Title), plain(r.Detail)
		if r.Title == "" || len(out.Risks) == MaxRisks {
			continue
		}
		r.Level = oneOf(r.Level, riskLevels, "medium")
		if !nodes[r.Node] {
			r.Node = ""
		}
		out.Risks = append(out.Risks, r)
	}
	out.Questions = plainList(bp.Questions, MaxQuestions)
	return out
}

// plain tira as crases que escaparem da regra de "nada de código no texto".
func plain(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "`", ""))
}

func plainList(items []string, limit int) []string {
	out := []string{}
	for _, s := range items {
		if s = plain(s); s != "" && len(out) < limit {
			out = append(out, s)
		}
	}
	return out
}

func oneOf(value string, allowed []string, fallback string) string {
	for _, a := range allowed {
		if value == a {
			return value
		}
	}
	return fallback
}
