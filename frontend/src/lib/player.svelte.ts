import { api, type MusicLibrary } from "./api";
import { AmbientMusic, playClick } from "./audio";

const SETTINGS_KEY = "oddeye:player";

interface Saved {
  volume: number;
  sfx: boolean;
  enabled: boolean;
  index: number;
}

/**
 * Estado único do som do app (um módulo, várias telas podem mostrar o controle).
 * Toca as músicas da pasta do usuário em sequência e em loop; sem músicas na pasta,
 * toca a trilha ambiente sintetizada.
 */
class Player {
  library = $state<MusicLibrary>({ dir: "", tracks: [] });
  index = $state(0);
  playing = $state(false);
  volume = $state(0.5);
  sfx = $state(true);
  enabled = $state(true);
  error = $state("");

  private audio: HTMLAudioElement | null = null;
  private objectURL: string | null = null;
  private failures = 0;
  private generation = 0;
  private ambient = new AmbientMusic();
  private started = false;

  get mode(): "folder" | "ambient" {
    return this.library.tracks.length ? "folder" : "ambient";
  }

  get title(): string {
    if (this.mode === "ambient") return "Trilha ambiente da galáxia";
    return (this.library.tracks[this.index] ?? "").replace(/\.[^.]+$/, "");
  }

  async init(): Promise<void> {
    try {
      const saved = JSON.parse(localStorage.getItem(SETTINGS_KEY) ?? "null") as Saved | null;
      if (saved) {
        this.volume = saved.volume;
        this.sfx = saved.sfx;
        this.enabled = saved.enabled;
        this.index = saved.index;
      }
    } catch {
      /* preferências indisponíveis: segue com o padrão */
    }
    await this.reload();

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

  async reload(): Promise<void> {
    try {
      this.library = await api.listMusic();
    } catch {
      this.library = { dir: "", tracks: [] };
    }
    if (this.index >= this.library.tracks.length) this.index = 0;
  }

  async play(): Promise<void> {
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
    const track = this.library.tracks[this.index];
    try {
      /* O player de mídia do WebKitGTK (Linux) recusa a URL do esquema próprio do Wails (wails://), embora
         o fetch dela funcione: a faixa é baixada pelo fetch e tocada a partir de uma URL blob:. */
      const response = await fetch(api.musicURL(track));
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
    } catch (e) {
      if (generation !== this.generation) return;
      this.fail(track, e instanceof Error ? e.message : String(e));
    }
    this.save();
  }

  /* Falha de uma faixa pula para a próxima; se todas falharam em sequência, para de pular (sem isso
     o player entra num loop trocando de faixa) e volta para a trilha ambiente, mostrando o erro. */
  private fail(track: string, reason: string): void {
    this.failures++;
    this.error = `Não consegui tocar "${track}": ${reason}.`;
    if (this.failures >= this.library.tracks.length) {
      this.failures = 0;
      this.stopAll();
      this.ambient.start(this.volume);
      this.playing = true;
      this.error += " Voltei para a trilha ambiente.";
      return;
    }
    this.index = (this.index + 1) % this.library.tracks.length;
    void this.play();
  }

  pause(): void {
    this.generation++;
    this.stopAll();
    this.playing = false;
    this.enabled = false;
    this.save();
  }

  toggle(): void {
    if (this.playing) this.pause();
    else void this.play();
  }

  next(): void {
    this.failures = 0;
    if (this.mode === "folder") this.index = (this.index + 1) % this.library.tracks.length;
    void this.play();
  }

  prev(): void {
    this.failures = 0;
    if (this.mode === "folder") this.index = (this.index - 1 + this.library.tracks.length) % this.library.tracks.length;
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
    this.index = 0;
    if (this.playing) void this.play();
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
      const s: Saved = { volume: this.volume, sfx: this.sfx, enabled: this.enabled, index: this.index };
      localStorage.setItem(SETTINGS_KEY, JSON.stringify(s));
    } catch {
      /* preferências são conveniência */
    }
  }
}

export const player = new Player();
