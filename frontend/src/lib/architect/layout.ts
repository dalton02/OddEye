/* Layout dos diagramas do modo Arquiteto. Funções puras (sem DOM), para dar para testar com bun. */

export interface GraphNode {
  id: string;
  kind: string;
}

export interface GraphEdge {
  from: string;
  to: string;
}

export interface Box {
  id: string;
  x: number;
  y: number;
  rank: number;
}

export interface Route {
  /** índice da aresta na lista de entrada */
  index: number;
  d: string;
  labelX: number;
  labelY: number;
}

export interface MapLayout {
  width: number;
  height: number;
  boxes: Box[];
  routes: Route[];
}

export const NODE_W = 178;
export const NODE_H = 66;
const COL_GAP = 150;
const ROW_GAP = 30;
const PAD = 28;
/* espaço acima das caixas para as arestas que voltam (ciclos) passarem por cima */
const TOP_ROOM = 34;

/* quem só recebe e guarda/entrega dados fica na ponta direita do mapa */
const SINKS = new Set(["store", "external"]);

/**
 * Posiciona os componentes em colunas, da esquerda para a direita, seguindo o sentido das ligações:
 * quem inicia fica à esquerda, bancos e serviços externos à direita. Dentro da coluna, a ordem segue
 * a média das posições dos vizinhos (baricentro), para as ligações cruzarem menos.
 */
export function layoutMap(nodes: GraphNode[], edges: GraphEdge[]): MapLayout {
  if (nodes.length === 0) return { width: 0, height: 0, boxes: [], routes: [] };
  const ids = nodes.map((n) => n.id);
  const kind = new Map(nodes.map((n) => [n.id, n.kind]));
  const out = new Map<string, string[]>(ids.map((id) => [id, []]));
  const valid = edges.filter((e) => out.has(e.from) && out.has(e.to) && e.from !== e.to);

  /* tira as arestas de retorno (ciclos) do cálculo de coluna com uma busca em profundidade */
  const forward: GraphEdge[] = [];
  for (const e of valid) out.get(e.from)!.push(e.to);
  const state = new Map<string, 0 | 1 | 2>();
  const back = new Set<string>();
  const visit = (id: string): void => {
    state.set(id, 1);
    for (const to of out.get(id)!) {
      if (state.get(to) === 1) back.add(id + "\u0000" + to);
      else if (!state.get(to)) visit(to);
    }
    state.set(id, 2);
  };
  /* começa pelos atores e por quem ninguém chama, para o início do fluxo ficar à esquerda */
  const called = new Set(valid.map((e) => e.to));
  const roots = [...ids.filter((id) => kind.get(id) === "actor"), ...ids.filter((id) => !called.has(id)), ...ids];
  for (const id of roots) if (!state.get(id)) visit(id);
  for (const e of valid) if (!back.has(e.from + "\u0000" + e.to)) forward.push(e);

  /* coluna = caminho mais longo até o nó */
  const rank = new Map<string, number>(ids.map((id) => [id, 0]));
  for (let pass = 0; pass < ids.length; pass++) {
    let moved = false;
    for (const e of forward) {
      const want = rank.get(e.from)! + 1;
      if (want > rank.get(e.to)!) {
        rank.set(e.to, want);
        moved = true;
      }
    }
    if (!moved) break;
  }
  const hasOut = new Set(forward.map((e) => e.from));
  const last = Math.max(...rank.values());
  for (const id of ids) if (SINKS.has(kind.get(id)!) && !hasOut.has(id) && last > 0) rank.set(id, last);

  const columns: string[][] = [];
  for (const id of ids) (columns[rank.get(id)!] ??= []).push(id);
  const cols = columns.filter((c) => c && c.length > 0);
  cols.forEach((col, r) => col.forEach((id) => rank.set(id, r)));

  /* baricentro: duas passadas de ida e volta bastam para grafos deste tamanho */
  const row = new Map<string, number>();
  cols.forEach((col) => col.forEach((id, i) => row.set(id, i)));
  const neighbors = new Map<string, string[]>(ids.map((id) => [id, []]));
  for (const e of valid) {
    neighbors.get(e.from)!.push(e.to);
    neighbors.get(e.to)!.push(e.from);
  }
  const sweep = (r: number, side: -1 | 1): void => {
    const score = (id: string): number => {
      const near = neighbors.get(id)!.filter((n) => rank.get(n) === r + side);
      if (near.length === 0) return row.get(id)!;
      return near.reduce((sum, n) => sum + row.get(n)!, 0) / near.length;
    };
    const scored = cols[r].map((id) => ({ id, s: score(id) }));
    scored.sort((a, b) => a.s - b.s);
    cols[r] = scored.map((x) => x.id);
    cols[r].forEach((id, i) => row.set(id, i));
  };
  for (let pass = 0; pass < 2; pass++) {
    for (let r = 1; r < cols.length; r++) sweep(r, -1);
    for (let r = cols.length - 2; r >= 0; r--) sweep(r, 1);
  }

  const tallest = Math.max(...cols.map((c) => c.length));
  const innerH = tallest * NODE_H + (tallest - 1) * ROW_GAP;
  const width = PAD * 2 + cols.length * NODE_W + (cols.length - 1) * COL_GAP;
  const height = PAD * 2 + TOP_ROOM + innerH;
  const boxes: Box[] = [];
  const at = new Map<string, Box>();
  cols.forEach((col, r) => {
    const colH = col.length * NODE_H + (col.length - 1) * ROW_GAP;
    const top = PAD + TOP_ROOM + (innerH - colH) / 2;
    col.forEach((id, i) => {
      const box = { id, x: PAD + r * (NODE_W + COL_GAP), y: top + i * (NODE_H + ROW_GAP), rank: r };
      boxes.push(box);
      at.set(id, box);
    });
  });

  const routes: Route[] = [];
  let bottom = height;
  let lanes = 0;
  edges.forEach((e, index) => {
    const a = at.get(e.from);
    const b = at.get(e.to);
    if (!a || !b || a === b) return;
    const between = boxes.filter((x) => x.rank > a.rank && x.rank < b.rank);
    const direct = route(a, b);
    if (between.length === 0 || !crosses(a, b, between)) {
      routes.push({ index, ...direct });
      return;
    }
    /* a ligação longa passaria por cima de uma caixa: desvia por um corredor acima ou abaixo delas */
    const top = Math.min(...between.map((x) => x.y));
    const low = Math.max(...between.map((x) => x.y + NODE_H));
    const mid = (a.y + b.y) / 2 + NODE_H / 2;
    const offset = 16 + lanes++ * 11;
    const y = mid - top < low - mid && top - offset > 10 ? top - offset : low + offset;
    bottom = Math.max(bottom, y + PAD);
    routes.push({ index, ...detour(a, b, y) });
  });
  return { width, height: bottom, boxes, routes };
}

const round = (n: number): number => Math.round(n * 10) / 10;

/** A curva direta de A até B passa por dentro de alguma caixa das colunas do meio? */
function crosses(a: Box, b: Box, between: Box[]): boolean {
  const x1 = a.x + NODE_W;
  const y1 = a.y + NODE_H / 2;
  const x2 = b.x;
  const y2 = b.y + NODE_H / 2;
  const bend = Math.min(COL_GAP * 0.6, (x2 - x1) / 2);
  const margin = 6;
  for (let i = 1; i < 40; i++) {
    const [x, y] = cubic(i / 40, [x1, y1], [x1 + bend, y1], [x2 - bend, y2], [x2, y2]);
    if (between.some((k) => x > k.x - margin && x < k.x + NODE_W + margin && y > k.y - margin && y < k.y + NODE_H + margin)) return true;
  }
  return false;
}

/** Sai de A, desce (ou sobe) no primeiro vão até a altura y, corre reto e entra em B pelo último vão. */
function detour(a: Box, b: Box, y: number): Omit<Route, "index"> {
  const x1 = a.x + NODE_W;
  const y1 = a.y + NODE_H / 2;
  const x2 = b.x;
  const y2 = b.y + NODE_H / 2;
  const half = COL_GAP / 2;
  const d =
    `M${round(x1)},${round(y1)} C${round(x1 + half)},${round(y1)} ${round(x1 + half)},${round(y)} ${round(x1 + COL_GAP)},${round(y)} ` +
    `L${round(x2 - COL_GAP)},${round(y)} C${round(x2 - half)},${round(y)} ${round(x2 - half)},${round(y2)} ${round(x2)},${round(y2)}`;
  return { d, labelX: round((x1 + x2) / 2), labelY: round(y) };
}

function route(a: Box, b: Box): Omit<Route, "index"> {
  if (b.rank > a.rank) {
    /* para a frente: sai da direita de A e entra na esquerda de B, com tangentes horizontais */
    const x1 = a.x + NODE_W;
    const y1 = a.y + NODE_H / 2;
    const x2 = b.x;
    const y2 = b.y + NODE_H / 2;
    const bend = Math.min(COL_GAP * 0.6, (x2 - x1) / 2);
    /* rótulo no primeiro vão entre colunas: numa ligação longa o meio cairia em cima de outra caixa */
    const t = 0.5 / (b.rank - a.rank);
    const [lx, ly] = cubic(t, [x1, y1], [x1 + bend, y1], [x2 - bend, y2], [x2, y2]);
    return {
      d: `M${round(x1)},${round(y1)} C${round(x1 + bend)},${round(y1)} ${round(x2 - bend)},${round(y2)} ${round(x2)},${round(y2)}`,
      labelX: round(lx),
      labelY: round(ly)
    };
  }
  if (a.rank === b.rank) {
    /* mesma coluna: curva pela lateral direita */
    const x = a.x + NODE_W;
    const y1 = a.y + NODE_H / 2;
    const y2 = b.y + NODE_H / 2;
    const bulge = 54;
    const [lx, ly] = cubic(0.5, [x, y1], [x + bulge, y1], [x + bulge, y2], [x, y2]);
    return {
      d: `M${round(x)},${round(y1)} C${round(x + bulge)},${round(y1)} ${round(x + bulge)},${round(y2)} ${round(x)},${round(y2)}`,
      labelX: round(lx),
      labelY: round(ly)
    };
  }
  /* para trás (ciclo): arco por cima de todas as caixas, de topo a topo, usando a folga TOP_ROOM */
  const x1 = a.x + NODE_W / 2 - 24;
  const x2 = b.x + NODE_W / 2 + 24;
  const top = PAD - 14 - Math.min(a.y, b.y) / 3;
  const [lx, ly] = cubic(0.5, [x1, a.y], [x1, top], [x2, top], [x2, b.y]);
  return {
    d: `M${round(x1)},${round(a.y)} C${round(x1)},${round(top)} ${round(x2)},${round(top)} ${round(x2)},${round(b.y)}`,
    labelX: round(lx),
    labelY: round(Math.max(12, ly))
  };
}

type Point = [number, number];

function cubic(t: number, p0: Point, p1: Point, p2: Point, p3: Point): Point {
  const u = 1 - t;
  const at = (i: 0 | 1): number => u * u * u * p0[i] + 3 * u * u * t * p1[i] + 3 * u * t * t * p2[i] + t * t * t * p3[i];
  return [at(0), at(1)];
}

/* ---------- Diagrama de sequência ---------- */

export interface SeqStep {
  from: string;
  to: string;
}

export interface Lifeline {
  id: string;
  x: number;
}

export interface SeqArrow {
  y: number;
  x1: number;
  x2: number;
  /** passo interno ao componente: vira um laço na própria linha de vida */
  self: boolean;
  /** onde o rótulo da ação é ancorado */
  labelX: number;
  anchor: "start" | "middle" | "end";
}

export interface SeqLayout {
  width: number;
  height: number;
  headH: number;
  lifelines: Lifeline[];
  arrows: SeqArrow[];
}

export const SEQ_HEAD_W = 150;
export const SEQ_HEAD_H = 44;
const SEQ_COL = 210;
const SEQ_ROW = 62;
const SEQ_PAD = 24;
/* largura reservada à direita para o rótulo de um laço na última linha de vida */
const SEQ_SELF_ROOM = 240;

/** Linhas de vida na ordem em que os componentes entram no fluxo; um passo por linha. */
export function layoutSequence(steps: SeqStep[]): SeqLayout {
  const order: string[] = [];
  for (const s of steps) for (const id of [s.from, s.to]) if (!order.includes(id)) order.push(id);
  const x = new Map(order.map((id, i) => [id, SEQ_PAD + SEQ_HEAD_W / 2 + i * SEQ_COL]));
  const lastSelf = steps.some((s) => s.from === s.to && s.from === order[order.length - 1]);
  const arrows = steps.map((s, i): SeqArrow => {
    const y = SEQ_PAD + SEQ_HEAD_H + 46 + i * SEQ_ROW;
    const x1 = x.get(s.from)!;
    const x2 = x.get(s.to)!;
    if (s.from === s.to) return { y, x1, x2, self: true, labelX: x1 + 44, anchor: "start" };
    return { y, x1, x2, self: false, labelX: (x1 + x2) / 2, anchor: "middle" };
  });
  return {
    width: SEQ_PAD * 2 + SEQ_HEAD_W + Math.max(0, order.length - 1) * SEQ_COL + (lastSelf ? SEQ_SELF_ROOM : 0),
    height: SEQ_PAD * 2 + SEQ_HEAD_H + 46 + Math.max(0, steps.length - 1) * SEQ_ROW + 30,
    headH: SEQ_HEAD_H,
    lifelines: order.map((id) => ({ id, x: x.get(id)! })),
    arrows
  };
}

/** Quebra um rótulo em até `lines` linhas de no máximo `max` caracteres, com reticências se sobrar. */
export function wrap(text: string, max: number, lines = 2): string[] {
  const out: string[] = [];
  let current = "";
  for (const word of text.split(/\s+/).filter(Boolean)) {
    const next = current ? current + " " + word : word;
    if (next.length <= max || !current) {
      current = next;
      continue;
    }
    out.push(current);
    current = word;
  }
  if (current) out.push(current);
  if (out.length > lines) {
    out.length = lines;
    out[lines - 1] = out[lines - 1].replace(/[\s.,;:]*$/, "") + "…";
  }
  return out.map((l) => (l.length > max + 1 ? l.slice(0, max) + "…" : l));
}
