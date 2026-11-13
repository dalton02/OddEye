import type { Blueprint, BlueprintNode, Change, ImpactAxisName, NodeKind } from "../api";

export const kindLabel: Record<NodeKind, string> = {
  actor: "quem usa",
  interface: "interface",
  service: "serviço",
  worker: "segundo plano",
  store: "dados",
  queue: "fila / evento",
  external: "externo"
};

export const changeLabel: Record<Change, string> = {
  new: "novo",
  changed: "alterado",
  removed: "removido",
  context: "contexto"
};

/* Cores de estado da mudança. Sempre acompanhadas do selo de texto: a cor nunca fala sozinha. */
export const changeColor: Record<Change, string> = {
  new: "#3fb950",
  changed: "#e3b341",
  removed: "#f85149",
  context: "#8f8f8f"
};

export const prKindLabel: Record<string, string> = {
  feature: "Funcionalidade nova",
  fix: "Correção",
  refactor: "Refatoração",
  performance: "Desempenho",
  security: "Segurança",
  infra: "Infraestrutura",
  chore: "Manutenção"
};

export const axisLabel: Record<ImpactAxisName, string> = {
  simplicity: "Simplicidade",
  reliability: "Confiabilidade",
  performance: "Desempenho",
  maintainability: "Manutenção",
  decoupling: "Desacoplamento",
  operability: "Operação"
};

export const levelLabel: Record<string, string> = { high: "Alto", medium: "Médio", low: "Baixo" };

export type ChapterKind = "overview" | "map" | "flow" | "decisions" | "impact" | "risks";

export interface Chapter {
  kind: ChapterKind;
  title: string;
  /** frase curta para o índice da visão geral */
  teaser: string;
  /** só para kind === "flow" */
  flow: number;
}

const plural = (n: number, one: string, many: string): string => `${n} ${n === 1 ? one : many}`;

/** Os capítulos do tour de arquitetura. Capítulo sem conteúdo não entra. */
export function chaptersOf(bp: Blueprint): Chapter[] {
  const out: Chapter[] = [{ kind: "overview", title: "Visão geral", teaser: "", flow: -1 }];
  if (bp.nodes.length) {
    const touched = bp.nodes.filter((n) => n.change !== "context").length;
    out.push({
      kind: "map",
      title: "Mapa do sistema",
      teaser: `${plural(bp.nodes.length, "componente", "componentes")}, ${touched} ${touched === 1 ? "tocado" : "tocados"} pela PR`,
      flow: -1
    });
  }
  bp.flows.forEach((f, i) =>
    out.push({ kind: "flow", title: f.title || `Fluxo ${i + 1}`, teaser: `Passo a passo em ${plural(f.steps.length, "etapa", "etapas")}`, flow: i })
  );
  if (bp.decisions.length) {
    out.push({
      kind: "decisions",
      title: "Decisões e trade-offs",
      teaser: `${plural(bp.decisions.length, "decisão", "decisões")} de desenho, com o que cada uma ganha e custa`,
      flow: -1
    });
  }
  if (bp.impact.length || bp.nodes.some((n) => n.weight.additions + n.weight.deletions > 0)) {
    out.push({ kind: "impact", title: "Impacto", teaser: "O que melhora, o que piora e onde está o peso da mudança", flow: -1 });
  }
  if (bp.risks.length || bp.questions.length) {
    out.push({
      kind: "risks",
      title: "Riscos e perguntas",
      teaser: `${plural(bp.risks.length, "risco", "riscos")} e ${plural(bp.questions.length, "pergunta", "perguntas")} para o autor`,
      flow: -1
    });
  }
  return out;
}

export interface NarrationState {
  /** passo atual do fluxo; -1 = visão do fluxo inteiro */
  step: number;
  selectedNode: string | null;
  chapters: number;
}

/** O que o mascote fala em cada parada do tour de arquitetura. */
export function narrate(bp: Blueprint, chapter: Chapter, state: NarrationState): string {
  switch (chapter.kind) {
    case "overview":
      if (state.chapters <= 1) return bp.summary;
      return `${bp.summary}\n\nMontei um tour em ${state.chapters - 1} capítulos para você entender a abordagem sem abrir o código. Use as setas para começar.`;
    case "map": {
      const node = bp.nodes.find((n) => n.id === state.selectedNode);
      if (node) return `${node.label}: ${node.role}`;
      return "Esse é o pedaço do sistema que a PR toca. As ligações seguem da esquerda para a direita, de quem inicia a ação até onde os dados ficam. Clique num componente para eu explicar o papel dele, ou alterne entre antes e depois para ver o que muda.";
    }
    case "flow": {
      const flow = bp.flows[chapter.flow];
      const step = flow?.steps[state.step];
      if (!flow) return "";
      if (!step) return `${flow.summary} Avance com a seta para percorrer o fluxo passo a passo.`;
      return `${step.action}. ${step.note}`.replace(/\.\.\s/, ". ");
    }
    case "decisions":
      return "Estas são as escolhas de desenho que a PR faz. Nenhuma sai de graça: veja o que cada uma ganha, o que custa e quais caminhos ficaram de fora.";
    case "impact":
      return impactNarration(bp);
    case "risks":
      if (!bp.risks.length) return "Não encontrei risco de arquitetura relevante. Ficaram algumas perguntas que valem a conversa com o autor: marque as que quiser levar para o review.";
      return `Separei ${plural(bp.risks.length, "ponto", "pontos")} de arquitetura para você ter em mente antes de aprovar. Marque as perguntas que quiser levar para o review.`;
  }
}

function impactNarration(bp: Blueprint): string {
  const sorted = [...bp.impact].sort((a, b) => b.score - a.score);
  const best = sorted[0];
  const worst = sorted[sorted.length - 1];
  const parts: string[] = [];
  if (best && best.score > 0) parts.push(`O maior ganho é em ${axisLabel[best.axis].toLowerCase()}: ${lowerFirst(best.note)}`);
  if (worst && worst.score < 0) parts.push(`O preço aparece em ${axisLabel[worst.axis].toLowerCase()}: ${lowerFirst(worst.note)}`);
  if (!parts.length) return "Essa PR é neutra para a arquitetura: não melhora nem piora de forma relevante nenhum dos eixos. Abaixo está onde o volume da mudança se concentra.";
  return parts.join("\n\n");
}

const lowerFirst = (s: string): string => (s ? s[0].toLowerCase() + s.slice(1) : s);

/** O componente dono de um arquivo da PR, para explicar um arquivo aberto a partir do mapa. */
export function ownerOf(bp: Blueprint, path: string): BlueprintNode | undefined {
  return bp.nodes.find((n) => n.files.includes(path));
}
