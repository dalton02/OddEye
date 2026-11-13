/* Regras puras da biblioteca de música (sem Svelte nem bindings), testadas em tests/music.test.ts. */

export interface Track {
  id: string;
  title: string;
  artist: string;
  source: "folder" | "torrent";
  name?: string;
  infoHash?: string;
  path?: string;
}

export interface TorrentFile {
  path: string;
  size: number;
  done: number;
  complete: boolean;
  wanted: boolean;
}

/** URL servida pelo backend para tocar a faixa (pasta local ou arquivo de torrent já baixado). */
export function trackURL(t: Track): string {
  if (t.source === "torrent") {
    return `/torrent/${t.infoHash}/${encodeURIComponent(t.path ?? "")}`;
  }
  return "/music/" + encodeURIComponent(t.name ?? "");
}

/** Identifica o arquivo para onde a faixa aponta; igual ao playlist.Key do Go. */
export function trackKey(t: Track): string {
  return t.source === "torrent" ? `t:${(t.infoHash ?? "").toLowerCase()}:${t.path}` : `f:${t.name}`;
}

export function baseTitle(path: string): string {
  const base = path.split(/[\\/]/).pop() ?? path;
  return base.replace(/\.[^.]+$/, "");
}

export function extOf(path: string): string {
  const m = /\.([^./\\]+)$/.exec(path);
  return m ? m[1].toLowerCase() : "";
}

export function folderTrack(name: string): Track {
  return { id: "f:" + name, title: baseTitle(name), artist: "", source: "folder", name };
}

export function torrentTrack(infoHash: string, path: string, artist = ""): Track {
  return { id: `t:${infoHash}:${path}`, title: baseTitle(path), artist, source: "torrent", infoHash, path };
}

/* Itens do archive.org trazem o mesmo álbum em vários formatos; mp3 é o mais leve que toca em todo lugar. */
const FORMAT_ORDER = ["mp3", "ogg", "opus", "m4a", "aac", "flac", "wav", "oga", "webm"];

/** Formatos presentes, do preferido para o menos preferido. */
export function formatsOf(files: TorrentFile[]): string[] {
  const present = new Set(files.map((f) => extOf(f.path)));
  const known = FORMAT_ORDER.filter((f) => present.has(f));
  const rest = [...present].filter((f) => !FORMAT_ORDER.includes(f)).sort();
  return [...known, ...rest];
}

/**
 * Arquivos de um formato, em ordem de caminho. No mp3, as cópias de 64 kbps (sufixo _64kb) só
 * aparecem quando não há versão melhor do mesmo arquivo.
 */
export function filesOfFormat(files: TorrentFile[], format: string): TorrentFile[] {
  const of = files.filter((f) => extOf(f.path) === format);
  const full = new Set(of.filter((f) => !/_64kb\.mp3$/i.test(f.path)).map((f) => f.path));
  return of
    .filter((f) => !/_64kb\.mp3$/i.test(f.path) || !full.has(f.path.replace(/_64kb\.mp3$/i, ".mp3")))
    .sort((a, b) => a.path.localeCompare(b.path, undefined, { numeric: true }));
}

export function fmtBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  const units = ["KB", "MB", "GB", "TB"];
  let v = n;
  let u = -1;
  do {
    v /= 1024;
    u++;
  } while (v >= 1024 && u < units.length - 1);
  return `${v < 10 ? v.toFixed(1) : Math.round(v)} ${units[u]}`;
}

export function percent(done: number, size: number): number {
  return size > 0 ? Math.min(100, Math.floor((done / size) * 100)) : 0;
}
