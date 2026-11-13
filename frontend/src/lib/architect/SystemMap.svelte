<script lang="ts">
  import type { BlueprintEdge, BlueprintNode, Change } from "../api";
  import { changeColor, changeLabel, kindLabel } from "./chapters";
  import { layoutMap, wrap, NODE_H, NODE_W } from "./layout";

  export type MapView = "before" | "diff" | "after";

  interface Props {
    nodes: BlueprintNode[];
    edges: BlueprintEdge[];
    selected: string | null;
    view: MapView;
    onopenfile: (path: string) => void;
  }

  let { nodes, edges, selected = $bindable(), view = $bindable(), onopenfile }: Props = $props();

  let hover = $state<string | null>(null);

  const layout = $derived(layoutMap(nodes, edges));
  const byId = $derived(new Map(nodes.map((n) => [n.id, n])));
  const current = $derived(nodes.find((n) => n.id === selected));

  const views: { value: MapView; label: string; hint: string }[] = [
    { value: "before", label: "Antes", hint: "Como o sistema era antes da PR" },
    { value: "diff", label: "O que muda", hint: "Tudo junto, com o que é novo, alterado e removido em destaque" },
    { value: "after", label: "Depois", hint: "Como o sistema fica com a PR" }
  ];

  /* Antes esconde o que a PR cria; Depois esconde o que ela remove. As posições não mudam entre as visões. */
  function gone(change: Change): boolean {
    return (view === "before" && change === "new") || (view === "after" && change === "removed");
  }
  const hidden = $derived(new Set(nodes.filter((n) => gone(n.change)).map((n) => n.id)));
  const focus = $derived.by(() => {
    const id = hover ?? selected;
    return id !== null && !hidden.has(id) ? id : null;
  });

  function show(next: MapView): void {
    view = next;
    /* o componente selecionado não existe nesse momento do sistema: a seleção não faz mais sentido */
    if (selected !== null && hidden.has(selected)) selected = null;
  }

  function edgeHidden(e: BlueprintEdge): boolean {
    return gone(e.change) || hidden.has(e.from) || hidden.has(e.to);
  }

  /* fora da visão "O que muda" o mapa é um retrato do sistema: ninguém aparece em destaque */
  function tone(change: Change): Change {
    return view === "diff" ? change : "context";
  }

  function edgeDimmed(e: BlueprintEdge): boolean {
    return focus !== null && e.from !== focus && e.to !== focus;
  }

  function nodeDimmed(id: string): boolean {
    if (focus === null || id === focus) return false;
    return !edges.some((e) => !edgeHidden(e) && ((e.from === focus && e.to === id) || (e.to === focus && e.from === id)));
  }

  const dash: Record<string, string> = { sync: "", async: "8 6", data: "2 6" };

  const changes: Change[] = ["new", "changed", "removed", "context"];
  const usedChanges = $derived(changes.filter((c) => nodes.some((n) => n.change === c) || edges.some((e) => e.change === c)));
  const usedStyles = $derived((["sync", "async", "data"] as const).filter((s) => edges.some((e) => e.style === s)));
  const styleLabel = { sync: "chamada direta", async: "evento / segundo plano", data: "leitura ou escrita de dados" };

  /* legenda numa linha só; as cores só entram na visão "O que muda", a única em que aparecem */
  const colorWord: Record<Change, string> = { new: "verde", changed: "amarelo", removed: "vermelho", context: "cinza" };
  const lineWord = { sync: "linha contínua", async: "linha tracejada", data: "linha pontilhada" };
  const legend = $derived.by(() => {
    const parts = [
      ...(view === "diff" ? usedChanges.map((c) => `${colorWord[c]}: ${changeLabel[c]}`) : []),
      ...usedStyles.map((s) => `${lineWord[s]}: ${styleLabel[s]}`)
    ].join(" · ");
    return parts ? parts[0].toUpperCase() + parts.slice(1) : "";
  });

  function toggle(id: string): void {
    selected = selected === id ? null : id;
  }

  function short(path: string): string {
    return path.split("/").pop() ?? path;
  }
</script>

<div class="map">
  <div class="bar">
    <div class="seg" role="group" aria-label="Momento do sistema">
      {#each views as v (v.value)}
        <button class:on={view === v.value} title={v.hint} aria-pressed={view === v.value} onclick={() => show(v.value)}>{v.label}</button>
      {/each}
    </div>
    {#if legend}<span class="label legend">{legend}</span>{/if}
  </div>

  <div class="canvas">
    <svg
      viewBox="0 0 {layout.width} {layout.height}"
      style="width: min(100%, {Math.round(layout.width * 1.3)}px); min-width: {Math.round(layout.width * 0.62)}px"
      role="group"
      aria-label="Mapa dos componentes do sistema e das ligações entre eles"
    >
      <defs>
        {#each changes as c (c)}
          <marker id="tip-{c}" viewBox="0 0 10 10" refX="8.5" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
            <path d="M1,1 L9,5 L1,9 z" fill={changeColor[c]} />
          </marker>
        {/each}
      </defs>

      {#each layout.routes as r (r.index)}
        {@const e = edges[r.index]}
        {@const t = tone(e.change)}
        <g class="edge" class:hidden={edgeHidden(e)} class:dim={edgeDimmed(e)} class:lit={focus !== null && !edgeDimmed(e)}>
          <path d={r.d} stroke={changeColor[t]} stroke-dasharray={dash[e.style]} marker-end="url(#tip-{t})" class:removed={t === "removed"} />
          {#if e.label}
            <text x={r.labelX} y={r.labelY - 6} text-anchor="middle">{wrap(e.label, 22, 1)[0]}</text>
          {/if}
        </g>
      {/each}

      {#each layout.boxes as b (b.id)}
        {@const n = byId.get(b.id)!}
        {@const t = tone(n.change)}
        {@const lines = wrap(n.label, 17, 2)}
        <g
          class="node"
          class:hidden={hidden.has(n.id)}
          class:dim={nodeDimmed(n.id)}
          class:selected={selected === n.id}
          transform="translate({b.x},{b.y})"
          role="button"
          tabindex={hidden.has(n.id) ? -1 : 0}
          aria-pressed={selected === n.id}
          aria-label="{n.label}, {kindLabel[n.kind]}, {changeLabel[n.change]}"
          onclick={() => toggle(n.id)}
          onkeydown={(ev) => {
            if (ev.key === "Enter" || ev.key === " ") {
              ev.preventDefault();
              toggle(n.id);
            }
          }}
          onpointerenter={() => (hover = n.id)}
          onpointerleave={() => (hover = null)}
        >
          <rect class="ring" x="-4" y="-4" width={NODE_W + 8} height={NODE_H + 8} rx="10" />
          <rect class="body" width={NODE_W} height={NODE_H} rx="6" stroke={changeColor[t]} class:removed={t === "removed"} class:ctx={t === "context"} />
          <text class="label" x="14" y={lines.length > 1 ? 24 : 30}>
            {#each lines as l, i (i)}<tspan x="14" dy={i === 0 ? 0 : 15}>{l}</tspan>{/each}
          </text>
          <text class="kind" x="14" y={lines.length > 1 ? 54 : 47}>
            {kindLabel[n.kind]}{t !== "context" ? " · " + changeLabel[t] : ""}
          </text>
        </g>
      {/each}
    </svg>
  </div>

  <div class="detail">
    {#if current}
      <span class="who">{current.label}</span>
      <span class="meta">{kindLabel[current.kind]} · {changeLabel[current.change]}</span>
      {#if current.files.length}
        <span class="meta">
          <b class="add">+{current.weight.additions}</b> <b class="del">−{current.weight.deletions}</b> em
          {current.files.length} arquivo{current.files.length === 1 ? "" : "s"}
        </span>
        <span class="files">
          {#each current.files as f (f)}
            <button class="link mono file" title="Abrir {f} no código" onclick={() => onopenfile(f)}>{short(f)}</button>
          {/each}
        </span>
      {:else}
        <span class="meta">sem arquivos nesta PR</span>
      {/if}
    {:else}
      <span class="meta">Clique num componente para ver o papel dele e os arquivos que o formam.</span>
    {/if}
  </div>
</div>

<style>
  .map {
    height: 100%;
    display: flex;
    flex-direction: column;
    gap: var(--sp-3);
    min-height: 0;
  }
  .bar {
    display: flex;
    align-items: center;
    gap: var(--sp-2) var(--sp-4);
    flex-wrap: wrap;
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

  .edge {
    transition: opacity var(--ease);
  }
  .edge path {
    fill: none;
    stroke-width: 1.5;
    stroke-linecap: round;
  }
  .edge path.removed {
    opacity: 0.6;
  }
  .edge text {
    font: var(--fs-xs) var(--ui);
    fill: var(--muted);
    /* contorno na cor do fundo: o rótulo fica legível mesmo cruzando outra linha */
    paint-order: stroke;
    stroke: var(--bg);
    stroke-width: 5px;
    stroke-linejoin: round;
  }
  .edge.lit path {
    stroke-width: 2;
    opacity: 1;
  }
  .edge.dim {
    opacity: 0.15;
  }

  .node {
    cursor: pointer;
    outline: none;
    transition: opacity var(--ease);
  }
  .node .body {
    fill: var(--surface);
    stroke-width: 1;
  }
  .node .body.ctx {
    stroke: var(--border-strong);
  }
  .node .body.removed {
    stroke-dasharray: 6 5;
  }
  .node:hover .body,
  .node:focus-visible .body {
    fill: var(--surface-2);
  }
  .node.selected .body {
    fill: var(--selected);
  }
  /* contorno branco do componente selecionado; no foco do teclado ele faz as vezes do outline */
  .node .ring {
    fill: none;
    stroke: var(--primary);
    stroke-width: 2;
    visibility: hidden;
  }
  .node.selected .ring,
  .node:focus-visible .ring {
    visibility: visible;
  }
  .node .label {
    font: 600 var(--fs-sm) var(--ui);
    fill: var(--text);
  }
  .node .kind {
    font: var(--fs-xs) var(--ui);
    fill: var(--muted);
  }
  .node.selected .kind {
    fill: var(--text-2);
  }
  .node.dim {
    opacity: 0.3;
  }
  .hidden {
    opacity: 0.07 !important;
    pointer-events: none;
  }

  .detail {
    display: flex;
    align-items: baseline;
    flex-wrap: wrap;
    gap: var(--sp-1) var(--sp-4);
    padding-top: var(--sp-3);
    border-top: 1px solid var(--border);
  }
  .who {
    font-weight: 600;
  }
  .meta {
    color: var(--muted);
  }
  .files {
    display: flex;
    flex-wrap: wrap;
    gap: var(--sp-1) var(--sp-3);
  }
  .file {
    font-size: var(--fs-xs);
  }
</style>
