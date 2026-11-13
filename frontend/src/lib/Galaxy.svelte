<script lang="ts">
  import { onMount } from "svelte";

  interface Props {
    dim?: boolean;
  }

  let { dim = false }: Props = $props();

  let backdrop: HTMLCanvasElement;
  let canvas: HTMLCanvasElement;

  interface Star {
    x: number;
    y: number;
    size: number;
    phase: number;
    speed: number;
    color: string;
    layer: number;
  }

  interface Meteor {
    x: number;
    y: number;
    vx: number;
    vy: number;
    life: number;
  }

  /* Custo por frame mantido constante e baixo: nebulosa e galáxia são pintadas uma vez em canvases
     fora da tela, e cada frame só faz um drawImage girado da galáxia, umas centenas de fillRect de
     estrelas e o meteoro. O loop roda a 30 fps e para quando a janela não está visível. */
  const FRAME_MS = 1000 / 30;

  onMount(() => {
    const ctx = canvas.getContext("2d")!;
    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    let width = 0;
    let height = 0;
    let dpr = 1;
    let stars: Star[] = [];
    let galaxy: HTMLCanvasElement | null = null;
    let galaxySize = 0;
    let meteor: Meteor | null = null;
    let mouseX = 0;
    let mouseY = 0;
    let frame = 0;
    let last = 0;

    const starColors = ["#ffffff", "#dfe8ff", "#fff4e0", "#cdd9ff", "#ffe2cf"];

    function rand(a: number, b: number): number {
      return a + Math.random() * (b - a);
    }

    function gauss(): number {
      return (Math.random() + Math.random() + Math.random() - 1.5) / 1.5;
    }

    function buildStars(): void {
      const area = width * height;
      stars = [];
      [0.00011, 0.00005, 0.00002].forEach((density, layer) => {
        for (let i = 0; i < Math.floor(area * density); i++) {
          stars.push({
            x: Math.random() * width,
            y: Math.random() * height,
            size: rand(0.6, 1.4) + layer * 0.7,
            phase: Math.random() * Math.PI * 2,
            speed: rand(0.4, 1.6),
            color: starColors[Math.floor(Math.random() * starColors.length)],
            layer
          });
        }
      });
    }

    /* Galáxia de frente (sem inclinação), pintada uma vez; a inclinação e a rotação vêm da transformação no drawImage. */
    function buildGalaxy(): void {
      galaxySize = Math.round(Math.min(width, height) * 1.25);
      const px = Math.min(galaxySize * dpr, 1800);
      galaxy = document.createElement("canvas");
      galaxy.width = px;
      galaxy.height = px;
      const g = galaxy.getContext("2d")!;
      const c = px / 2;
      const r = px / 2;

      const core = g.createRadialGradient(c, c, 0, c, c, r * 0.45);
      core.addColorStop(0, "rgba(255, 238, 215, 0.9)");
      core.addColorStop(0.12, "rgba(255, 200, 160, 0.45)");
      core.addColorStop(0.4, "rgba(150, 120, 220, 0.12)");
      core.addColorStop(1, "rgba(120, 100, 200, 0)");
      g.fillStyle = core;
      g.fillRect(0, 0, px, px);

      g.globalCompositeOperation = "lighter";
      const arms = 2;
      const count = 14000;
      const dot = Math.max(1, px / 900);
      for (let i = 0; i < count; i++) {
        const radius = Math.pow(Math.random(), 1.5);
        const angle = (i % arms) * ((Math.PI * 2) / arms) + radius * 4.2 + (0.5 + radius * 0.6) * gauss() * 0.7;
        const jitter = 0.02 + radius * 0.07;
        const x = c + (Math.cos(angle) * radius + gauss() * jitter) * r * 0.95;
        const y = c + (Math.sin(angle) * radius + gauss() * jitter) * r * 0.95;
        let color: string;
        if (radius < 0.12) color = `255, ${Math.floor(rand(210, 240))}, ${Math.floor(rand(170, 205))}`;
        else if (Math.random() < 0.12) color = `255, ${Math.floor(rand(120, 170))}, ${Math.floor(rand(170, 210))}`;
        else color = `${Math.floor(rand(150, 200))}, ${Math.floor(rand(175, 215))}, 255`;
        g.fillStyle = `rgba(${color}, ${(1 - radius) * rand(0.25, 0.85)})`;
        const s = (radius < 0.1 ? rand(0.8, 1.8) : rand(0.4, 1.3)) * dot;
        g.fillRect(x, y, s, s);
      }
    }

    /* Fundo estático (preto + nebulosa) num canvas próprio, desenhado só no resize. */
    function paintBackdrop(): void {
      backdrop.width = width * dpr;
      backdrop.height = height * dpr;
      const b = backdrop.getContext("2d")!;
      b.setTransform(dpr, 0, 0, dpr, 0, 0);
      b.fillStyle = "#030407";
      b.fillRect(0, 0, width, height);
      const blobs: [number, number, number, string][] = [
        [0.15, 0.2, 0.45, "88, 60, 170"],
        [0.85, 0.75, 0.5, "40, 80, 170"],
        [0.55, 0.95, 0.4, "150, 50, 120"],
        [0.05, 0.85, 0.35, "30, 110, 150"],
        [0.7, 0.1, 0.3, "120, 70, 200"]
      ];
      for (const [x, y, r, c] of blobs) {
        const g = b.createRadialGradient(x * width, y * height, 0, x * width, y * height, r * Math.max(width, height));
        g.addColorStop(0, `rgba(${c}, 0.11)`);
        g.addColorStop(0.5, `rgba(${c}, 0.035)`);
        g.addColorStop(1, `rgba(${c}, 0)`);
        b.fillStyle = g;
        b.fillRect(0, 0, width, height);
      }
    }

    function resize(): void {
      dpr = Math.min(window.devicePixelRatio || 1, 1.5);
      width = window.innerWidth;
      height = window.innerHeight;
      canvas.width = width * dpr;
      canvas.height = height * dpr;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      paintBackdrop();
      buildStars();
      buildGalaxy();
    }

    function drawGalaxy(time: number): void {
      if (!galaxy) return;
      ctx.save();
      ctx.translate(width * 0.72 + mouseX * 10, height * 0.42 + mouseY * 8);
      ctx.rotate(-0.42);
      ctx.scale(1, 0.42);
      ctx.rotate(reduced ? 0 : time * 0.00003);
      ctx.globalCompositeOperation = "lighter";
      ctx.drawImage(galaxy, -galaxySize / 2, -galaxySize / 2, galaxySize, galaxySize);
      ctx.restore();
    }

    function drawStars(time: number): void {
      const t = time * 0.001;
      for (const s of stars) {
        const drift = reduced ? 0 : time * 0.0015 * (s.layer + 1);
        const x = (((s.x - drift + mouseX * (s.layer + 1) * 6) % width) + width) % width;
        const y = (((s.y + mouseY * (s.layer + 1) * 4) % height) + height) % height;
        const tw = reduced ? 0.8 : 0.55 + 0.45 * Math.sin(t * s.speed + s.phase);
        ctx.globalAlpha = tw;
        ctx.fillStyle = s.color;
        ctx.fillRect(x, y, s.size, s.size);
        if (s.layer === 2 && tw > 0.9) {
          ctx.globalAlpha = (tw - 0.9) * 4;
          const h = s.size / 2;
          ctx.fillRect(x + h - s.size * 3, y + h - 0.3, s.size * 6, 0.6);
          ctx.fillRect(x + h - 0.3, y + h - s.size * 3, 0.6, s.size * 6);
        }
      }
      ctx.globalAlpha = 1;
    }

    function drawMeteor(): void {
      if (reduced) return;
      if (!meteor && Math.random() < 0.004) {
        meteor = { x: rand(width * 0.2, width), y: rand(0, height * 0.4), vx: -rand(14, 22), vy: rand(5, 9), life: 1 };
      }
      if (!meteor) return;
      const tail = 9;
      const g = ctx.createLinearGradient(meteor.x, meteor.y, meteor.x - meteor.vx * tail, meteor.y - meteor.vy * tail);
      g.addColorStop(0, `rgba(255, 255, 255, ${meteor.life})`);
      g.addColorStop(1, "rgba(255, 255, 255, 0)");
      ctx.strokeStyle = g;
      ctx.lineWidth = 1.6;
      ctx.beginPath();
      ctx.moveTo(meteor.x, meteor.y);
      ctx.lineTo(meteor.x - meteor.vx * tail, meteor.y - meteor.vy * tail);
      ctx.stroke();
      meteor.x += meteor.vx;
      meteor.y += meteor.vy;
      meteor.life -= 0.024;
      if (meteor.life <= 0 || meteor.x < -200 || meteor.y > height + 200) meteor = null;
    }

    function render(time: number): void {
      frame = requestAnimationFrame(render);
      if (time - last < FRAME_MS) return;
      last = time;
      ctx.clearRect(0, 0, width, height);
      drawStars(time);
      drawGalaxy(time);
      drawMeteor();
    }

    function onMouse(e: MouseEvent): void {
      mouseX = e.clientX / width - 0.5;
      mouseY = e.clientY / height - 0.5;
    }

    function onVisibility(): void {
      cancelAnimationFrame(frame);
      if (!document.hidden) frame = requestAnimationFrame(render);
    }

    let resizeTimer: ReturnType<typeof setTimeout> | undefined;
    function onResize(): void {
      clearTimeout(resizeTimer);
      resizeTimer = setTimeout(resize, 120);
    }

    resize();
    frame = requestAnimationFrame(render);
    window.addEventListener("resize", onResize);
    window.addEventListener("mousemove", onMouse, { passive: true });
    document.addEventListener("visibilitychange", onVisibility);
    return () => {
      cancelAnimationFrame(frame);
      clearTimeout(resizeTimer);
      window.removeEventListener("resize", onResize);
      window.removeEventListener("mousemove", onMouse);
      document.removeEventListener("visibilitychange", onVisibility);
    };
  });
</script>

<div class="sky" class:dim aria-hidden="true">
  <canvas bind:this={backdrop}></canvas>
  <canvas bind:this={canvas}></canvas>
</div>

<style>
  .sky {
    position: fixed;
    inset: 0;
    z-index: 0;
    pointer-events: none;
    background: #030407;
    transition: opacity 0.6s ease;
  }
  /* Durante a leitura o céu fica em segundo plano: menos brilho e movimento na visão periférica. */
  .sky.dim {
    opacity: 0.45;
  }
  canvas {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    display: block;
  }
</style>
