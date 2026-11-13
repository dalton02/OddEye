import { api, errorMessage, type AudioStatus, type MusicLibrary, type Playlist } from "./api";
import { AmbientMusic, playClick } from "./audio";
import { folderTrack, percent, trackURL, type Track } from "./music";

const SETTINGS_KEY = "oddeye:player";
const POLL_MS = 1000;

interface Saved {
  volume: number;
  sfx: boolean;
  enabled: boolean;
  index: number;
  /** playlist tocando ao fechar o app; "" é a pasta local */
  playlistId?: string;
}

/**
 * Estado único do som do app (um módulo, várias telas podem mostrar o controle).
 * Toca uma fila de faixas em sequência e em loop: a pasta local, uma playlist ou um álbum aberto
 * na busca. Faixa de torrent que ainda não baixou é baixada com prioridade antes de tocar. Sem
 * nenhuma faixa na fila, toca a trilha ambiente sintetizada.
 */
class Player {
  library = $state<MusicLibrary>({ dir: "", tracks: [] });
  playlists = $state<Playlist[]>([]);
  queue = $state<Track[]>([]);
  /** de onde a fila veio: id de playlist, "" para a pasta local ou null para uma fila avulsa (álbum da busca) */
  playlistId = $state<string | null>("");
  queueLabel = $state("Pasta local");
  index = $state(0);
  playing = $state(false);
  /** texto de espera enquanto a faixa de torrent baixa ("Baixando 42%") */
  loading = $state("");
  volume = $state(0.5);
  sfx = $state(true);
  enabled = $state(true);
  error = $state("");
  libraryOpen = $state(false);
  /** preenchido quando o sistema não consegue tocar som: o player fica desligado e explica o que falta */
  unavailable = $state<AudioStatus | null>(null);

  private audio: HTMLAudioElement | null = null;
  private objectURL: string | null = null;
  private failures = 0;
  private generation = 0;
  private ambient = new AmbientMusic();
  private started = false;

  get mode(): "queue" | "ambient" {
    return this.queue.length ? "queue" : "ambient";
  }

  get current(): Track | null {
    return this.queue[this.index] ?? null;
  }

  get title(): string {
    if (this.mode === "ambient") return "Trilha ambiente";
    return this.current?.title ?? "";
  }

  async init(): Promise<void> {
    try {
      const saved = JSON.parse(localStorage.getItem(SETTINGS_KEY) ?? "null") as Saved | null;
      if (saved) {
        this.volume = saved.volume;
        this.sfx = saved.sfx;
        this.enabled = saved.enabled;
        this.index = saved.index;
        this.playlistId = saved.playlistId ?? "";
      }
    } catch {
      /* preferências indisponíveis: segue com o padrão */
    }
    await this.reload();

    /* Sem saída de som no sistema, nenhum áudio é criado: cada tentativa só geraria erro no log do webview.
       Se a consulta falhar, segue com o som ligado, que é o comportamento de sempre. */
    try {
      const status = await api.audioStatus();
      if (!status.available) {
        this.unavailable = status;
        return;
      }
    } catch {
      /* binding indisponível: assume que há som */
    }

    /* o webview só libera áudio depois de um gesto: a música começa no primeiro clique em qualquer lugar */
    const kick = (): void => {
      if (!this.started && this.enabled) void this.play();
    };
    window.addEventListener("pointerdown", kick, { once: true });
    window.addEventListener("keydown", kick, { once: true });

    document.addEventListener(
      "click",
      (e) => {
        const target = e.target as HTMLElement | null;
        if (this.sfx && target?.closest("button, .ref, label, [role=button]")) playClick(this.volume);
      },
      true
    );
  }

  /** Relê a pasta e as playlists. A fila atual é refeita a partir delas, mantendo a faixa tocando. */
  async reload(): Promise<void> {
    try {
      this.library = await api.listMusic();
    } catch {
      this.library = { dir: "", tracks: [] };
    }
    try {
      this.playlists = await api.listPlaylists();
    } catch {
      this.playlists = [];
    }
    if (this.playlistId === null) return; // fila avulsa: não vem de nenhuma das duas
    const list = this.playlistId ? this.playlists.find((p) => p.id === this.playlistId) : undefined;
    if (this.playlistId && !list) this.playlistId = ""; // playlist apagada: volta para a pasta
    const tracks = list ? list.tracks : this.library.tracks.map(folderTrack);
    this.replaceQueue(tracks, list ? list.name : "Pasta local");
  }

  /** Troca a fila sem interromper a faixa atual quando ela continua presente. */
  private replaceQueue(tracks: Track[], label: string): void {
    const cur = this.current;
    this.queue = tracks;
    this.queueLabel = label;
    const keep = cur ? tracks.findIndex((t) => t.id === cur.id) : -1;
    if (keep >= 0) this.index = keep;
    else if (this.index >= tracks.length) this.index = 0;
  }

  /** Toca uma fila a partir da posição start. playlistId null marca uma fila avulsa. */
  playQueue(tracks: Track[], start: number, label: string, playlistId: string | null): void {
    this.queue = tracks;
    this.queueLabel = label;
    this.playlistId = playlistId;
    this.index = Math.max(0, Math.min(start, tracks.length - 1));
    this.failures = 0;
    void this.play();
  }

  playPlaylist(id: string, start = 0): void {
    if (id === "") {
      this.playQueue(this.library.tracks.map(folderTrack), start, "Pasta local", "");
      return;
    }
    const list = this.playlists.find((p) => p.id === id);
    if (list?.tracks.length) this.playQueue(list.tracks, start, list.name, id);
  }

  /** Chamado pela biblioteca depois de mexer numa playlist: a fila acompanha se for a mesma. */
  playlistChanged(list: Playlist): void {
    const i = this.playlists.findIndex((p) => p.id === list.id);
    if (i >= 0) this.playlists[i] = list;
    else this.playlists.push(list);
    if (this.playlistId === list.id) this.replaceQueue(list.tracks, list.name);
  }

  playlistDeleted(id: string): void {
    this.playlists = this.playlists.filter((p) => p.id !== id);
    if (this.playlistId === id) {
      this.playlistId = "";
      this.replaceQueue(this.library.tracks.map(folderTrack), "Pasta local");
    }
  }

  async play(): Promise<void> {
    if (this.unavailable) return;
    this.started = true;
    this.enabled = true;
    this.stopAll();
    const generation = ++this.generation;
    if (this.mode === "ambient") {
      this.ambient.start(this.volume);
      this.playing = true;
      this.save();
      return;
    }
    const track = this.queue[this.index];
    /* mostra pausar já durante a espera do download, para o clique seguinte cancelar */
    this.playing = true;
    try {
      if (track.source === "torrent") await this.ensureDownloaded(track, generation);
      if (generation !== this.generation) return;
      /* O player de mídia do WebKitGTK (Linux) recusa a URL do esquema próprio do Wails (wails://), embora
         o fetch dela funcione: a faixa é baixada pelo fetch e tocada a partir de uma URL blob:. */
      const response = await fetch(trackURL(track));
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const blob = await response.blob();
      if (generation !== this.generation) return;
      this.objectURL = URL.createObjectURL(blob);
      const audio = new Audio(this.objectURL);
      audio.volume = this.volume * 0.6;
      audio.onended = () => this.next();
      audio.onplaying = () => {
        this.failures = 0;
        this.error = "";
      };
      audio.onerror = () => this.fail(track, `o formato não é suportado (código ${audio.error?.code ?? "?"})`);
      this.audio = audio;
      await audio.play();
      this.playing = true;
      this.prefetchNext();
    } catch (e) {
      if (generation !== this.generation) return;
      this.fail(track, errorMessage(e));
    } finally {
      if (generation === this.generation) this.loading = "";
    }
    this.save();
  }

  /* Pede o arquivo com prioridade e espera terminar. Outra faixa escolhida no meio (generation
     diferente) encerra a espera; o download segue em segundo plano. */
  private async ensureDownloaded(track: Track, generation: number): Promise<void> {
    const hash = track.infoHash ?? "";
    const path = track.path ?? "";
    let file = await api.downloadTrack(hash, path, true);
    while (!file.complete) {
      if (generation !== this.generation) return;
      this.loading = `Baixando ${percent(file.done, file.size)}%`;
      await new Promise((r) => setTimeout(r, POLL_MS));
      const files = await api.torrentFiles(hash);
      const f = files.find((x) => x.path === path);
      if (!f) throw new Error("o arquivo sumiu do torrent");
      file = f;
    }
  }

  /** Começa a baixar a próxima faixa de torrent enquanto esta toca, para a troca não ter espera. */
  private prefetchNext(): void {
    const next = this.queue[(this.index + 1) % this.queue.length];
    if (next?.source === "torrent" && next !== this.current) {
      void api.downloadTrack(next.infoHash ?? "", next.path ?? "", false).catch(() => {});
    }
  }

  /* Falha de uma faixa pula para a próxima; se todas falharam em sequência, para de pular (sem isso
     o player entra num loop trocando de faixa) e volta para a trilha ambiente, mostrando o erro. */
  private fail(track: Track, reason: string): void {
    this.failures++;
    this.loading = "";
    this.error = `Não consegui tocar "${track.title}": ${reason}.`;
    if (this.failures >= this.queue.length) {
      this.failures = 0;
      this.stopAll();
      this.ambient.start(this.volume);
      this.playing = true;
      this.error += " Voltei para a trilha ambiente.";
      return;
    }
    this.index = (this.index + 1) % this.queue.length;
    void this.play();
  }

  pause(): void {
    this.generation++;
    this.stopAll();
    this.playing = false;
    this.loading = "";
    this.enabled = false;
    this.save();
  }

  toggle(): void {
    if (this.playing) this.pause();
    else void this.play();
  }

  next(): void {
    this.failures = 0;
    if (this.mode === "queue") this.index = (this.index + 1) % this.queue.length;
    void this.play();
  }

  prev(): void {
    this.failures = 0;
    if (this.mode === "queue") this.index = (this.index - 1 + this.queue.length) % this.queue.length;
    void this.play();
  }

  setVolume(v: number): void {
    this.volume = v;
    if (this.audio) this.audio.volume = v * 0.6;
    this.ambient.setVolume(v);
    this.save();
  }

  toggleSfx(): void {
    this.sfx = !this.sfx;
    this.save();
  }

  async chooseFolder(): Promise<void> {
    const lib = await api.chooseMusicFolder();
    if (!lib) return;
    this.library = lib;
    if (this.playlistId === "") {
      this.queue = lib.tracks.map(folderTrack);
      this.index = 0;
      if (this.playing) void this.play();
    }
  }

  private stopAll(): void {
    this.ambient.stop();
    if (this.audio) {
      this.audio.onended = null;
      this.audio.onerror = null;
      this.audio.onplaying = null;
      this.audio.pause();
      this.audio = null;
    }
    if (this.objectURL) {
      URL.revokeObjectURL(this.objectURL);
      this.objectURL = null;
    }
  }

  private save(): void {
    try {
      const s: Saved = {
        volume: this.volume,
        sfx: this.sfx,
        enabled: this.enabled,
        index: this.index,
        /* fila avulsa não sobrevive ao reinício: volta para a pasta */
        playlistId: this.playlistId ?? ""
      };
      localStorage.setItem(SETTINGS_KEY, JSON.stringify(s));
    } catch {
      /* preferências são conveniência */
    }
  }
}

export const player = new Player();
