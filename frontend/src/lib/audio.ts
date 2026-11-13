/* Todo o som do app é sintetizado com Web Audio: nada de arquivo de áudio embutido. O contexto só
   nasce no primeiro gesto do usuário, porque o webview bloqueia áudio antes disso. */

let ctx: AudioContext | null = null;

export function audioContext(): AudioContext {
  if (!ctx) ctx = new AudioContext();
  if (ctx.state === "suspended") void ctx.resume();
  return ctx;
}

/** Clique suave de botão: um "plim" senoidal curto, com filtro passa-baixa para não ficar estridente. */
export function playClick(volume: number): void {
  if (volume <= 0) return;
  const ac = audioContext();
  const now = ac.currentTime;
  const out = ac.createGain();
  const filter = ac.createBiquadFilter();
  filter.type = "lowpass";
  filter.frequency.value = 2200;
  out.connect(filter).connect(ac.destination);
  out.gain.setValueAtTime(0, now);
  out.gain.linearRampToValueAtTime(0.06 * volume, now + 0.006);
  out.gain.exponentialRampToValueAtTime(0.0001, now + 0.22);

  for (const [freq, level] of [
    [784, 1],
    [1568, 0.18]
  ]) {
    const osc = ac.createOscillator();
    const g = ac.createGain();
    osc.type = "sine";
    osc.frequency.setValueAtTime(freq, now);
    osc.frequency.exponentialRampToValueAtTime(freq * 0.94, now + 0.2);
    g.gain.value = level;
    osc.connect(g).connect(out);
    osc.start(now);
    osc.stop(now + 0.25);
  }
}

const midi = (n: number): number => 440 * Math.pow(2, (n - 69) / 12);

/* Progressão e escala próprias, escolhidas só pelo clima calmo (Ré maior, acordes com sétima e nona). */
const CHORDS = [
  [50, 57, 61, 64, 66],
  [47, 54, 57, 62, 66],
  [43, 50, 54, 59, 62],
  [45, 52, 57, 59, 64]
];
const MELODY = [74, 76, 78, 81, 83, 86, 88];

/**
 * Trilha ambiente original e infinita: pads lentos com notas de sino esparsas por cima,
 * passando por um eco com realimentação para dar espaço. Usada quando não há músicas na pasta.
 */
export class AmbientMusic {
  private master: GainNode | null = null;
  private timers: ReturnType<typeof setTimeout>[] = [];
  private chord = 0;
  private running = false;

  start(volume: number): void {
    if (this.running) return;
    this.running = true;
    const ac = audioContext();
    this.master = ac.createGain();
    this.master.gain.value = 0;
    this.master.gain.linearRampToValueAtTime(volume * 0.5, ac.currentTime + 3);

    const delay = ac.createDelay(2);
    delay.delayTime.value = 0.42;
    const feedback = ac.createGain();
    feedback.gain.value = 0.38;
    const tone = ac.createBiquadFilter();
    tone.type = "lowpass";
    tone.frequency.value = 1400;
    this.master.connect(ac.destination);
    this.master.connect(delay);
    delay.connect(tone).connect(feedback).connect(delay);
    tone.connect(ac.destination);

    this.playChord();
    this.scheduleBell();
  }

  setVolume(volume: number): void {
    if (!this.master) return;
    const ac = audioContext();
    this.master.gain.cancelScheduledValues(ac.currentTime);
    this.master.gain.linearRampToValueAtTime(volume * 0.5, ac.currentTime + 0.3);
  }

  stop(): void {
    this.running = false;
    this.timers.forEach(clearTimeout);
    this.timers = [];
    if (this.master) {
      const ac = audioContext();
      const m = this.master;
      m.gain.cancelScheduledValues(ac.currentTime);
      m.gain.linearRampToValueAtTime(0, ac.currentTime + 1.2);
      setTimeout(() => m.disconnect(), 1500);
      this.master = null;
    }
  }

  private playChord(): void {
    if (!this.running || !this.master) return;
    const ac = audioContext();
    const now = ac.currentTime;
    const length = 9;
    const filter = ac.createBiquadFilter();
    filter.type = "lowpass";
    filter.frequency.value = 900;
    const env = ac.createGain();
    env.gain.setValueAtTime(0, now);
    env.gain.linearRampToValueAtTime(0.09, now + 3);
    env.gain.setValueAtTime(0.09, now + length - 2);
    env.gain.linearRampToValueAtTime(0, now + length + 2.5);
    filter.connect(env).connect(this.master);
    for (const note of CHORDS[this.chord]) {
      for (const detune of [-6, 6]) {
        const osc = ac.createOscillator();
        osc.type = "triangle";
        osc.frequency.value = midi(note);
        osc.detune.value = detune;
        const g = ac.createGain();
        g.gain.value = 0.18;
        osc.connect(g).connect(filter);
        osc.start(now);
        osc.stop(now + length + 3);
      }
    }
    this.chord = (this.chord + 1) % CHORDS.length;
    this.timers.push(setTimeout(() => this.playChord(), length * 1000));
  }

  private scheduleBell(): void {
    if (!this.running) return;
    const wait = 1400 + Math.random() * 2600;
    this.timers.push(
      setTimeout(() => {
        if (Math.random() < 0.7) this.bell(MELODY[Math.floor(Math.random() * MELODY.length)]);
        this.scheduleBell();
      }, wait)
    );
  }

  private bell(note: number): void {
    if (!this.master) return;
    const ac = audioContext();
    const now = ac.currentTime;
    const env = ac.createGain();
    env.gain.setValueAtTime(0, now);
    env.gain.linearRampToValueAtTime(0.07, now + 0.01);
    env.gain.exponentialRampToValueAtTime(0.0001, now + 3.2);
    env.connect(this.master);
    for (const [ratio, level] of [
      [1, 1],
      [2, 0.25],
      [3.01, 0.08]
    ]) {
      const osc = ac.createOscillator();
      osc.type = "sine";
      osc.frequency.value = midi(note) * ratio;
      const g = ac.createGain();
      g.gain.value = level;
      osc.connect(g).connect(env);
      osc.start(now);
      osc.stop(now + 3.3);
    }
  }
}
