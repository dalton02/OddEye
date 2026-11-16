/* Lógica pura da tela Coding (testada em tests/coding.test.ts). */

export type Shortcut = "editor" | "new" | "shell" | "close" | "save" | "next" | "home" | { tab: number };

/** Atalho da tela Coding para a tecla, ou null. Sempre com Ctrl (Cmd no macOS) e sem Alt. Com Shift,
 *  só Ctrl+Shift+T (aba de shell) e Ctrl+Shift+Tab (troca a aba da home); Ctrl+Shift+C/V ficam com o xterm. */
export function shortcutFor(
  e: Pick<KeyboardEvent, "key" | "ctrlKey" | "metaKey" | "altKey" | "shiftKey"> & { code?: string }
): Shortcut | null {
  if (!(e.ctrlKey || e.metaKey) || e.altKey) return null;
  const k = e.key.toLowerCase();
  /* no WebKitGTK o Shift+Tab chega como ISO_Left_Tab; o code (tecla física) é sempre "Tab" */
  if (e.code === "Tab" || k === "tab" || k === "iso_left_tab") return e.shiftKey ? "home" : "next";
  if (e.shiftKey) return k === "t" ? "shell" : null;
  if (k === "e") return "editor";
  if (k === "t") return "new";
  if (k === "w") return "close";
  if (k === "s") return "save";
  if (/^[1-9]$/.test(k)) return { tab: Number(k) - 1 };
  return null;
}

/** Nome curto da pasta para o título da aba. */
export function folderName(dir: string): string {
  const parts = dir.replace(/[\\/]+$/, "").split(/[\\/]/);
  return parts[parts.length - 1] || dir;
}

/** Caminho para mostrar na aba: a pasta pessoal (Linux, macOS, Windows) vira "~". */
export function shortPath(dir: string): string {
  const m = /^(\/home\/[^/]+|\/Users\/[^/]+|[A-Za-z]:\\Users\\[^\\]+)(?=$|[\\/])/.exec(dir);
  return m ? "~" + dir.slice(m[0].length) : dir;
}

export function baseName(path: string): string {
  return path.split("/").pop() ?? path;
}

export function dirName(path: string): string {
  const i = path.lastIndexOf("/");
  return i < 0 ? "" : path.slice(0, i);
}

/** Pastas que contêm algum arquivo alterado, para marcar a árvore. */
export function changedDirs(paths: string[]): Set<string> {
  const out = new Set<string>();
  for (const p of paths) {
    let d = dirName(p);
    while (d) {
      out.add(d);
      d = dirName(d);
    }
  }
  return out;
}

export function decodeBase64(b64: string): Uint8Array {
  const bin = atob(b64);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}

/** Índice da aba seguinte a i, voltando à primeira depois da última. */
export function nextIndex(len: number, i: number): number {
  return len ? (i + 1) % len : 0;
}

export type HomeTab = "prs" | "board" | "coding";

/** Aba da home seguinte (Ctrl+Shift+Tab): Pull requests → Board → Coding → Pull requests. */
export function nextHomeTab(t: HomeTab): HomeTab {
  return t === "prs" ? "board" : t === "board" ? "coding" : "prs";
}

/** Próxima aba ativa depois de fechar a do índice i (fica a da direita, ou a última). */
export function nextActive<T>(tabs: T[], closed: number): T | null {
  const rest = tabs.filter((_, j) => j !== closed);
  return rest[Math.min(closed, rest.length - 1)] ?? null;
}
