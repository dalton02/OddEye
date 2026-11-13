<script lang="ts">
  import { onMount } from "svelte";

  /* Fundo da home e da análise: poucas estrelas em pixel art andando devagar para a esquerda.
     É só textura: nada pisca, nada segue o mouse, e com "reduzir movimento" o céu fica parado. */

  let canvas: HTMLCanvasElement;

  interface Star {
    x: number;
    y: number;
    speed: number;
    alpha: number;
    cross: boolean;
  }

  const PIXEL = 2;
  const DENSITY = 0.00003;
  /* o céu anda no máximo 5 px/s: 12 quadros por segundo já dão movimento contínuo */
  const FRAME_MS = 1000 / 12;

  onMount(() => {
    const ctx = canvas.getContext("2d")!;
    const motion = window.matchMedia("(prefers-reduced-motion: reduce)");
    let stars: Star[] = [];
    let width = 0;
    let height = 0;
    let frame = 0;
    let last = 0;

    function seed(): void {
      const count = Math.round(width * height * DENSITY);
      stars = Array.from({ length: count }, () => ({
        x: Math.random() * width,
        y: Math.random() * height,
        speed: 2 + Math.random() * 3,
        alpha: 0.25 + Math.random() * 0.45,
        cross: Math.random() < 0.2
      }));
    }

    function resize(): void {
      const dpr = window.devicePixelRatio || 1;
      width = window.innerWidth;
      height = window.innerHeight;
      canvas.width = Math.round(width * dpr);
      canvas.height = Math.round(height * dpr);
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      seed();
      draw();
    }

    function draw(): void {
      ctx.clearRect(0, 0, width, height);
      ctx.fillStyle = "#ededed";
      for (const s of stars) {
        /* posição presa à grade de pixels para a estrela não borrar entre dois pixels */
        const x = Math.round(s.x / PIXEL) * PIXEL;
        const y = Math.round(s.y / PIXEL) * PIXEL;
        ctx.globalAlpha = s.alpha;
        ctx.fillRect(x, y, PIXEL, PIXEL);
        if (s.cross) {
          ctx.fillRect(x - PIXEL, y, PIXEL, PIXEL);
          ctx.fillRect(x + PIXEL, y, PIXEL, PIXEL);
          ctx.fillRect(x, y - PIXEL, PIXEL, PIXEL);
          ctx.fillRect(x, y + PIXEL, PIXEL, PIXEL);
        }
      }
      ctx.globalAlpha = 1;
    }

    function tick(now: number): void {
      frame = requestAnimationFrame(tick);
      const elapsed = now - last;
      if (elapsed < FRAME_MS) return;
      last = now;
      /* aba que voltou de segundo plano: não dá um salto com o tempo acumulado */
      const dt = Math.min(elapsed, 250) / 1000;
      for (const s of stars) {
        s.x -= s.speed * dt;
        if (s.x < -PIXEL * 2) {
          s.x = width + PIXEL * 2;
          s.y = Math.random() * height;
        }
      }
      draw();
    }

    function sync(): void {
      cancelAnimationFrame(frame);
      if (motion.matches || document.hidden) return;
      last = performance.now();
      frame = requestAnimationFrame(tick);
    }

    let resizeTimer: ReturnType<typeof setTimeout> | undefined;
    function onResize(): void {
      clearTimeout(resizeTimer);
      resizeTimer = setTimeout(resize, 120);
    }

    resize();
    sync();
    window.addEventListener("resize", onResize);
    document.addEventListener("visibilitychange", sync);
    motion.addEventListener("change", sync);
    return () => {
      cancelAnimationFrame(frame);
      clearTimeout(resizeTimer);
      window.removeEventListener("resize", onResize);
      document.removeEventListener("visibilitychange", sync);
      motion.removeEventListener("change", sync);
    };
  });
</script>

<canvas bind:this={canvas} aria-hidden="true"></canvas>

<style>
  canvas {
    position: fixed;
    inset: 0;
    z-index: 0;
    width: 100%;
    height: 100%;
    display: block;
    pointer-events: none;
    image-rendering: pixelated;
  }
</style>
