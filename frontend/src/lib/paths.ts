export type Resolver = (token: string) => string | null;

const EXTENSIONS = [".ts", ".tsx", ".js", ".jsx", ".mjs", ".go", ".py", ".rb", ".java", ".kt", ".rs", ".cs", ".php", ".svelte", ".vue"];

/**
 * Cria o resolvedor que transforma uma citação de arquivo do Claude no caminho real do repositório.
 * Aceita o caminho completo, caminho de import com alias (`@infra/x` casa com `src/infra/x.ts`),
 * import sem extensão ou apontando para pasta (`.../x/index.ts`) e sufixo de caminho, desde que
 * o resultado seja único. Nome solto sem barra (`index.ts`) só casa quando não há ambiguidade.
 * @returns null quando a citação não corresponde a exatamente um arquivo.
 */
export function createResolver(paths: string[]): Resolver {
  const all = new Set(paths);
  const cache = new Map<string, string | null>();

  function candidates(base: string): string[] {
    const list = [base];
    for (const ext of EXTENSIONS) list.push(base + ext, base + "/index" + ext);
    return list;
  }

  function bySuffix(suffix: string): string | null {
    let found: string | null = null;
    for (const p of paths) {
      if (p === suffix || p.endsWith("/" + suffix)) {
        if (found && found !== p) return null;
        found = p;
      }
    }
    return found;
  }

  function resolve(raw: string): string | null {
    let token = raw.trim().replace(/^["']|["']$/g, "").replace(/^\.?\//, "").replace(/:\d+(-\d+)?$/, "");
    if (!token || /\s/.test(token) || token.length > 300) return null;
    if (!token.includes("/") && !/\.\w+$/.test(token)) return null;

    for (const c of candidates(token)) if (all.has(c)) return c;

    /* alias de import: `@infra/x` ou `~/x` → o resto do caminho é procurado como sufixo */
    if (/^[@~]/.test(token)) token = token.replace(/^[@~][^/]*\//, (m) => (m === "~/" ? "" : m.slice(1)));
    for (const c of candidates(token)) {
      const hit = bySuffix(c);
      if (hit) return hit;
    }
    return null;
  }

  return (token: string): string | null => {
    if (!cache.has(token)) cache.set(token, resolve(token));
    return cache.get(token)!;
  };
}

export interface TextPart {
  text: string;
  code: boolean;
  path: string | null;
}

/** Quebra o texto em trechos: código entre crases e caminhos soltos viram partes próprias, com link quando resolvem. */
export function parseRichText(text: string, resolve: Resolver): TextPart[] {
  const parts: TextPart[] = [];
  const re = /`([^`\n]+)`|((?:[\w@~.-]+\/)+[\w.-]+\.\w+)/g;
  let last = 0;
  for (const m of text.matchAll(re)) {
    const index = m.index ?? 0;
    if (index > last) parts.push({ text: text.slice(last, index), code: false, path: null });
    const inner = m[1] ?? m[2];
    parts.push({ text: inner, code: m[1] !== undefined, path: resolve(inner) });
    last = index + m[0].length;
  }
  if (last < text.length) parts.push({ text: text.slice(last), code: false, path: null });
  return parts;
}
