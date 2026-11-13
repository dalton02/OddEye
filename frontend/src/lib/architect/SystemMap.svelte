<script lang="ts">
  import type { BlueprintEdge, BlueprintNode, Change, NodeKind } from "../api";
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

  const icons: Record<NodeKind, string> = {
    actor: "M9 8a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM3 16c0-3.3 2.7-5 6-5s6 1.7 6 5",
    interface: "M2 4h14v11H2zM2 7.5h14",
    service: "M9 2l6 3.5v7L9 16l-6-3.5v-7z",
    worker: "M14.5 9a5.5 5.5 0 1 1-1.6-3.9M13.2 1.8v3.5H9.7",
    store: "M3 5c0-1.4 2.7-2.5 6-2.5s6 1.1 6 2.5v8c0 1.4-2.7 2.5-6 2.5S3 14.4 3 13zM3 5c0 1.4 2.7 2.5 6 2.5s6-1.1 6-2.5",
    queue: "M2 6h3.5v6H2zM7.25 6h3.5v6h-3.5zM12.5 6H16v6h-3.5z",
    external: "M5.5 14a3.5 3.5 0 0 1-.4-7 4.5 4.5 0 0 1 8.7 1.2A3 3 0 0 1 13 14z"
  };

  const changes: Change[] = ["new", "changed", "removed", "context"];
  const usedChanges = $derived(changes.filter((c) => nodes.some((n) => n.change === c) || edges.some((e) => e.change === c)));
  const usedStyles = $derived((["sync", "async", "data"] as const).filter((s) => edges.some((e) => e.style === s)));
  const styleLabel = { sync: "chamada direta", async: "evento / segundo plano", data: "leitura ou escrita de dados" };

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
    <div class="legend">
      {#if view === "diff"}
        {#each usedChanges as c (c)}
          <span class="key"><i class="sw" class:dashed={c === "removed"} style="border-color: {changeColor[c]}"></i>{changeLabel[c]}</span>
        {/each}
      {/if}
      {#each usedStyles as s (s)}
        <span class="key">
          <svg width="26" height="8" aria-hidden="true"><line x1="1" y1="4" x2="25" y2="4" stroke-dasharray={dash[s]} /></svg>{styleLabel[s]}
        </span>
      {/each}
    </div>
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
          <rect class="body" width={NODE_W} height={NODE_H} rx="13" stroke={changeColor[t]} class:removed={t === "removed"} class:ctx={t === "context"} />
          {#if t !== "context"}
            <rect class="stripe" x="0" y="14" width="3.5" height={NODE_H - 28} rx="1.75" fill={changeColor[t]} />
          {/if}
          <path class="icon" d={icons[n.kind]} transform="translate(15,{NODE_H / 2 - 9})" />
          <text class="label" x="44" y={lines.length > 1 ? 24 : 30}>
            {#each lines as l, i (i)}<tspan x="44" dy={i === 0 ? 0 : 15}>{l}</tspan>{/each}
          </text>
          <text class="kind" x="44" y={lines.length > 1 ? 54 : 47}>
            {kindLabel[n.kind]}{t !== "context" ? " · " + changeLabel[t] : ""}
          </text>
        </g>
      {/each}
    </svg>
  </div>

  <div class="detail">
    {#if current}
      <span class="who"><i class="pip" style="background: {changeColor[current.change]}"></i>{current.label}</span>
      <span class="meta">{kindLabel[current.kind]} · {changeLabel[current.change]}</span>
      {#if current.files.length}
        <span class="meta">
          <b class="add">+{current.weight.additions}</b> <b class="del">−{current.weight.deletions}</b> em
          {current.files.length} arquivo{current.files.length === 1 ? "" : "s"}
        </span>
        <span class="files">
          {#each current.files as f (f)}
            <button class="file" title="Abrir {f} no código" onclick={() => onopenfile(f)}>{short(f)}</button>
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
    gap: 10px;
    min-height: 0;
  }
  .bar {
    display: flex;
    align-items: center;
    gap: 16px;
    flex-wrap: wrap;
  }
  .seg {
    display: inline-flex;
    padding: 3px;
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 11px;
  }
  .seg button {
    border: none;
    background: none;
    color: var(--muted);
    padding: 5px 13px;
    border-radius: 8px;
    font-size: 12.5px;
  }
  .seg button:hover {
    color: var(--text);
  }
  .seg button.on {
    background: rgba(217, 119, 87, 0.2);
    color: var(--text);
  }
  .legend {
    display: flex;
    flex-wrap: wrap;
    gap: 6px 14px;
    margin-left: auto;
    font-size: 11.5px;
    color: var(--muted);
  }
  .key {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
  .sw {
    width: 14px;
    height: 10px;
    border-radius: 3px;
    border: 1.5px solid;
  }
  .sw.dashed {
    border-style: dashed;
  }
  .key line {
    stroke: var(--muted);
    stroke-width: 1.6;
    stroke-linecap: round;
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

  .edge path {
    fill: none;
    stroke-width: 1.7;
    stroke-linecap: round;
    opacity: 0.85;
    transition: opacity 0.2s, stroke-width 0.2s;
  }
  .edge path.removed {
    opacity: 0.6;
  }
  .edge text {
    font: 11.5px var(--ui);
    fill: #c3c8d6;
    /* contorno na cor do painel: o rótulo fica legível mesmo cruzando outra linha */
    paint-order: stroke;
    stroke: #111217;
    stroke-width: 5px;
    stroke-linejoin: round;
    transition: opacity 0.2s;
  }
  .edge.lit path {
    stroke-width: 2.4;
    opacity: 1;
  }
  .edge.dim path,
  .edge.dim text {
    opacity: 0.12;
  }

  .node {
    cursor: pointer;
    outline: none;
    transition: opacity 0.25s;
  }
  .node .body {
    fill: #191b24;
    stroke-width: 1.5;
    transition: fill 0.15s, stroke-width 0.15s;
  }
  .node .body.ctx {
    stroke: rgba(160, 170, 210, 0.3);
  }
  .node .body.removed {
    stroke-dasharray: 6 5;
  }
  .node:hover .body,
  .node:focus-visible .body {
    fill: #20232f;
  }
  .node.selected .body {
    fill: #252838;
    stroke-width: 2.5;
  }
  .node.selected .body.ctx {
    stroke: var(--claude-soft);
  }
  .node:focus-visible .body {
    stroke-width: 2.5;
  }
  .node .icon {
    fill: none;
    stroke: #aab1c6;
    stroke-width: 1.5;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .node .label {
    font: 700 13px var(--ui);
    fill: var(--text);
  }
  .node .kind {
    font: 11px var(--ui);
    fill: var(--muted);
  }
  .node.dim {
    opacity: 0.3;
  }
  .hidden {
    opacity: 0.07 !important;
    pointer-events: none;
  }
  .edge {
    transition: opacity 0.25s;
  }

  .detail {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 6px 14px;
    min-height: 30px;
    font-size: 12.5px;
  }
  .who {
    display: inline-flex;
    align-items: center;
    gap: 7px;
    font-weight: 700;
  }
  .pip {
    width: 9px;
    height: 9px;
    border-radius: 50%;
  }
  .meta {
    color: var(--muted);
  }
  .add {
    color: var(--green);
    font-weight: 400;
  }
  .del {
    color: var(--red);
    font-weight: 400;
  }
  .files {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
  .file {
    background: rgba(91, 140, 214, 0.1);
    border: 1px solid rgba(91, 140, 214, 0.3);
    color: #b9d0f3;
    border-radius: 999px;
    padding: 2px 10px;
    font: 11.5px var(--mono);
  }
  .file:hover {
    background: rgba(91, 140, 214, 0.2);
  }
</style>
