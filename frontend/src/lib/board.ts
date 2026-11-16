// Lógica pura da aba Board (sem Svelte nem Wails), para ser testada com o bun.

export interface BoardOption {
  id: string;
  name: string;
  description: string;
  color: string;
}

export interface BoardField {
  id: string;
  name: string;
  options: BoardOption[];
}

export type CardKind = "issue" | "pr" | "draft";

export interface BoardCard {
  itemId: string;
  kind: CardKind;
  contentId: string;
  number: number;
  title: string;
  url: string;
  state: string;
  repo: string;
  assignees: string[];
  statusId: string;
  typeId: string;
  subTotal: number;
  subDone: number;
}

export interface Board {
  projectId: string;
  owner: string;
  number: number;
  title: string;
  url: string;
  public: boolean;
  status: BoardField;
  type: BoardField | null;
  cards: BoardCard[];
  repos: string[];
  truncated: boolean;
}

export interface Column {
  id: string; // opção do Status; "" = sem status
  name: string;
  description: string;
  cards: BoardCard[];
}

export interface CardFilter {
  text: string;
  typeId: string; // "" = todos
  onlyMine: boolean;
  viewer: string;
}

export function matches(c: BoardCard, f: CardFilter): boolean {
  if (f.typeId && c.typeId !== f.typeId) return false;
  if (f.onlyMine && !c.assignees.some((a) => a.toLowerCase() === f.viewer.toLowerCase())) return false;
  const q = f.text.trim().toLowerCase().replace(/^#/, "");
  if (!q) return true;
  return c.title.toLowerCase().includes(q) || String(c.number) === q || c.repo.toLowerCase().includes(q);
}

/** Uma coluna por opção do Status, na ordem do projeto; "Sem status" só aparece se tiver card. */
export function columns(board: Board, f: CardFilter): Column[] {
  const cols: Column[] = board.status.options.map((o) => ({ id: o.id, name: o.name, description: o.description, cards: [] }));
  const none: Column = { id: "", name: "Sem status", description: "Cards sem valor no campo Status", cards: [] };
  const byId = new Map(cols.map((c) => [c.id, c]));
  for (const card of board.cards) {
    if (!matches(card, f)) continue;
    (byId.get(card.statusId) ?? none).cards.push(card);
  }
  return none.cards.length ? [none, ...cols] : cols;
}

/** Cor com significado para a bolinha da coluna: bloqueado em vermelho, concluído em verde, o resto neutro. */
export function columnTone(name: string): "blocked" | "done" | "" {
  const n = name.toLowerCase();
  if (/block|bloque|imped/.test(n)) return "blocked";
  if (/^(done|conclu|feito|entregue|closed)/.test(n)) return "done";
  return "";
}

export function slug(text: string): string {
  return text
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

const MAX_BRANCH = 60;

/** Prefixo da branch pelo tipo do card: bug/, us/, task/... (sem tipo: feat/). */
export function branchPrefix(typeName: string): string {
  return slug(typeName) || "feat";
}

/** Nome de branch a partir do card: `task/1290-estudo-integracao-catalogo-meta`, cortado numa palavra inteira. */
export function branchName(typeName: string, number: number | null, title: string): string {
  const head = `${branchPrefix(typeName)}/${number ?? ""}`;
  let rest = slug(title);
  const room = MAX_BRANCH - head.length - (number == null ? 0 : 1);
  if (rest.length > room) {
    rest = rest.slice(0, room);
    const cut = rest.lastIndexOf("-");
    if (cut > room / 2) rest = rest.slice(0, cut);
    rest = rest.replace(/-+$/, "");
  }
  if (!rest) return head;
  return number == null ? head + rest : `${head}-${rest}`;
}

export function initials(login: string): string {
  const parts = login.split(/[-_.\s]+/).filter(Boolean);
  const s = parts.length > 1 ? parts[0][0] + parts[1][0] : login.slice(0, 2);
  return s.toUpperCase();
}

/** Move o card para outra coluna (cópia), para a atualização otimista. */
export function moveCard(board: Board, itemId: string, statusId: string): Board {
  return { ...board, cards: board.cards.map((c) => (c.itemId === itemId ? { ...c, statusId } : c)) };
}
