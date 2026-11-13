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

  /* participantes do passo atual ficam acesos no cabeçalho */
  const active = $derived(step >= 0 && flow.steps[step] ? new Set([flow.steps[step].from, flow.steps[step].to]) : null);
</script>

<div class="flow">
  <div class="bar">
    <div class="steps" role="group" aria-label="Passos do fluxo">
      <button class="pill all" class:on={step < 0} onclick={() => (step = -1)}>Fluxo inteiro</button>
      {#each flow.steps as s, i (i)}
        <button class="pill" class:on={step === i} class:done={step > i} title={s.action} aria-label="Passo {i + 1}: {s.action}" onclick={() => (step = i)}>
          {i + 1}
        </button>
      {/each}
    </div>
    <div class="legend">
      {#each changes as c (c)}
        <span class="key"><i style="background: {changeColor[c]}"></i>{stepLabel[c]}</span>
      {/each}
    </div>
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
          <rect x={l.x - SEQ_HEAD_W / 2} y="24" width={SEQ_HEAD_W} height={SEQ_HEAD_H} rx="11" />
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
          <rect class="hit" x="0" y={a.y - 34} width={layout.width} height="58" rx="10" />
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
    gap: 10px;
    min-height: 0;
  }
  .bar {
    display: flex;
    align-items: center;
    gap: 16px;
    flex-wrap: wrap;
  }
  .steps {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
  }
  .pill {
    min-width: 30px;
    height: 30px;
    padding: 0 9px;
    border-radius: 999px;
    border: 1px solid var(--border);
    background: var(--panel);
    color: var(--muted);
    font-size: 12.5px;
  }
  .pill:hover {
    color: var(--text);
    border-color: rgba(217, 119, 87, 0.5);
  }
  .pill.done {
    color: var(--text);
  }
  .pill.on {
    background: rgba(217, 119, 87, 0.22);
    border-color: rgba(217, 119, 87, 0.7);
    color: var(--text);
  }
  .legend {
    display: flex;
    gap: 14px;
    margin-left: auto;
    font-size: 11.5px;
    color: var(--muted);
  }
  .key {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
  .key i {
    width: 14px;
    height: 3px;
    border-radius: 2px;
  }

  .canvas {
    flex: 1;
    min-height: 0;
    overflow: auto;
    /* flex + margin auto centraliza sem cortar o topo quando o diagrama é maior que a área */
    display: flex;
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 6px;
  }
  svg {
    display: block;
    height: auto;
    margin: auto;
    flex-shrink: 0;
  }

  .life {
    transition: opacity 0.25s;
  }
  .life line {
    stroke: rgba(160, 170, 210, 0.22);
    stroke-width: 1;
  }
  .life rect {
    fill: #191b24;
    stroke: rgba(160, 170, 210, 0.35);
    stroke-width: 1.5;
  }
  .life text {
    font: 700 12.5px var(--ui);
    fill: var(--text);
  }
  .life.off {
    opacity: 0.45;
  }

  .step {
    cursor: pointer;
    outline: none;
    transition: opacity 0.25s;
  }
  .hit {
    fill: transparent;
  }
  .step:hover .hit,
  .step:focus-visible .hit {
    fill: rgba(255, 255, 255, 0.035);
  }
  .step.current .hit {
    fill: rgba(217, 119, 87, 0.1);
  }
  .arrow {
    fill: none;
    stroke-width: 2;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .step.current .arrow {
    stroke-width: 3;
  }
  .action {
    font: 12.5px var(--ui);
    fill: var(--text);
    paint-order: stroke;
    stroke: #111217;
    stroke-width: 5px;
    stroke-linejoin: round;
  }
  .step.current .action {
    font-weight: 700;
  }
  .num-bg {
    fill: #111217;
    stroke-width: 1.5;
  }
  .num {
    font: 700 11px var(--ui);
    fill: var(--text);
    pointer-events: none;
  }
  .step.future {
    opacity: 0.28;
  }
</style>
