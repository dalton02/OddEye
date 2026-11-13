<script lang="ts">
  import type { BlueprintNode, Change, Flow } from "../api";
  import { changeColor, changeLabel } from "./chapters";
  import { layoutSequence, wrap, SEQ_HEAD_H, SEQ_HEAD_W } from "./layout";

  interface Props {
    flow: Flow;
    nodes: BlueprintNode[];
    /** passo em destaque; -1 mostra o fluxo inteiro */
    step: number;
  }

  let { flow, nodes, step = $bindable() }: Props = $props();

  const layout = $derived(layoutSequence(flow.steps));
  const label = $derived(new Map(nodes.map((n) => [n.id, n.label])));
  const changes = $derived((["new", "changed", "context"] as Change[]).filter((c) => flow.steps.some((s) => s.change === c)));
  const stepLabel: Record<string, string> = { new: "passo novo", changed: "passo alterado", context: "já existia" };
  const colorWord: Record<string, string> = { new: "verde", changed: "amarelo", context: "cinza" };
  const legend = $derived.by(() => {
    const parts = changes.map((c) => `${colorWord[c]}: ${stepLabel[c]}`).join(" · ");
    return parts ? parts[0].toUpperCase() + parts.slice(1) : "";
  });

  /* participantes do passo atual ficam acesos no cabeçalho */
  const active = $derived(step >= 0 && flow.steps[step] ? new Set([flow.steps[step].from, flow.steps[step].to]) : null);
</script>

<div class="flow">
  <div class="bar">
    <button class="link" class:quiet={step < 0} onclick={() => (step = -1)}>Fluxo inteiro</button>
    {#if step >= 0}<span class="label">Passo {step + 1} de {flow.steps.length}</span>{/if}
    {#if legend}<span class="label legend">{legend}</span>{/if}
  </div>

  <div class="canvas">
    <svg
      viewBox="0 0 {layout.width} {layout.height}"
      style="width: min(100%, {Math.round(layout.width * 1.3)}px); min-width: {Math.round(layout.width * 0.62)}px"
      role="group"
      aria-label="Diagrama de sequência do fluxo {flow.title}"
    >
      <defs>
        {#each ["new", "changed", "context"] as c (c)}
          <marker id="seq-{c}" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="8" markerHeight="8" orient="auto-start-reverse">
            <path d="M1,1 L9,5 L1,9 z" fill={changeColor[c as Change]} />
          </marker>
        {/each}
      </defs>

      {#each layout.lifelines as l (l.id)}
        {@const lines = wrap(label.get(l.id) ?? l.id, 18, 2)}
        <g class="life" class:off={active !== null && !active.has(l.id)}>
          <line x1={l.x} y1={24 + SEQ_HEAD_H} x2={l.x} y2={layout.height - 16} />
          <rect x={l.x - SEQ_HEAD_W / 2} y="24" width={SEQ_HEAD_W} height={SEQ_HEAD_H} rx="6" />
          <text x={l.x} y={lines.length > 1 ? 42 : 50} text-anchor="middle">
            {#each lines as t, i (i)}<tspan x={l.x} dy={i === 0 ? 0 : 14}>{t}</tspan>{/each}
          </text>
        </g>
      {/each}

      {#each layout.arrows as a, i (i)}
        {@const s = flow.steps[i]}
        {@const dir = a.x2 >= a.x1 ? 1 : -1}
        <g
          class="step"
          class:current={step === i}
          class:future={step >= 0 && i > step}
          role="button"
          tabindex="0"
          aria-label="Passo {i + 1}: {s.action} ({changeLabel[s.change]})"
          onclick={() => (step = i)}
          onkeydown={(ev) => {
            if (ev.key === "Enter" || ev.key === " ") {
              ev.preventDefault();
              step = i;
            }
          }}
        >
          <rect class="hit" x="0" y={a.y - 34} width={layout.width} height="58" rx="6" />
          {#if a.self}
            <path class="arrow" d="M{a.x1},{a.y - 12} h30 v24 h-24" stroke={changeColor[s.change]} marker-end="url(#seq-{s.change})" />
            <text class="action" x={a.labelX} y={a.y + 4}>{wrap(s.action, 36, 1)[0]}</text>
          {:else}
            <line class="arrow" x1={a.x1 + dir * 13} y1={a.y} x2={a.x2 - dir * 3} y2={a.y} stroke={changeColor[s.change]} marker-end="url(#seq-{s.change})" />
            <text class="action" x={a.labelX + dir * 6} y={a.y - 9} text-anchor="middle">{wrap(s.action, 40, 1)[0]}</text>
          {/if}
          <circle class="num-bg" cx={a.x1} cy={a.self ? a.y - 12 : a.y} r="11" stroke={changeColor[s.change]} />
          <text class="num" x={a.x1} y={(a.self ? a.y - 12 : a.y) + 4} text-anchor="middle">{i + 1}</text>
        </g>
      {/each}
    </svg>
  </div>
</div>

<style>
  .flow {
    height: 100%;
    display: flex;
    flex-direction: column;
    gap: var(--sp-3);
    min-height: 0;
  }
  .bar {
    display: flex;
    align-items: baseline;
    gap: var(--sp-2) var(--sp-4);
    flex-wrap: wrap;
    min-height: 28px;
  }
  .legend {
    margin-left: auto;
  }

  .canvas {
    flex: 1;
    min-height: 0;
    overflow: auto;
    /* flex + margin auto centraliza sem cortar o topo quando o diagrama é maior que a área */
    display: flex;
  }
  svg {
    display: block;
    height: auto;
    margin: auto;
    flex-shrink: 0;
  }

  .life {
    transition: opacity var(--ease);
  }
  .life line {
    stroke: var(--border-strong);
    stroke-width: 1;
  }
  .life rect {
    fill: var(--surface);
    stroke: var(--border-strong);
    stroke-width: 1;
  }
  .life text {
    font: 600 var(--fs-sm) var(--ui);
    fill: var(--text);
  }
  .life.off {
    opacity: 0.45;
  }

  .step {
    cursor: pointer;
    outline: none;
    transition: opacity var(--ease);
  }
  .hit {
    fill: transparent;
  }
  .step:hover .hit,
  .step:focus-visible .hit {
    fill: var(--surface);
  }
  /* o foco do teclado contorna a faixa do passo, já que o outline não vale para <g> */
  .step:focus-visible .hit {
    stroke: var(--primary);
    stroke-width: 2;
  }
  .arrow {
    fill: none;
    stroke-width: 1.5;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .step.current .arrow {
    stroke-width: 2.5;
  }
  .action {
    font: var(--fs-sm) var(--ui);
    fill: var(--muted);
    paint-order: stroke;
    stroke: var(--bg);
    stroke-width: 5px;
    stroke-linejoin: round;
  }
  .step.current .action {
    font-weight: 600;
    fill: var(--primary);
  }
  .num-bg {
    fill: var(--bg);
    stroke-width: 1;
  }
  .num {
    font: 600 var(--fs-xs) var(--ui);
    fill: var(--text);
    pointer-events: none;
  }
  .step.future {
    opacity: 0.28;
  }
</style>
