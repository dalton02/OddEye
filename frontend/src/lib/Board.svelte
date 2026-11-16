<script lang="ts">
  import { onMount } from "svelte";
  import { api, errorMessage, type BoardPrefs, type CreatedCard, type ProjectSummary } from "./api";
  import { columns, columnTone, initials, moveCard, type Board, type BoardCard } from "./board";
  import CardPanel from "./CardPanel.svelte";
  import NewCardModal from "./NewCardModal.svelte";

  interface Props {
    /** repositórios vigiados nas Configurações: entram nas opções de repositório do card novo */
    watchedRepos: string[];
  }

  let { watchedRepos }: Props = $props();

  const PAGE = 40; // cards por coluna antes do "Mostrar mais"

  let prefs = $state<BoardPrefs>({ owner: "", number: 0, lastRepo: "", hidden: [] });
  let viewer = $state("");
  let board = $state<Board | null>(null);
  let loading = $state(false);
  let error = $state("");
  let notice = $state("");

  // escolha do projeto
  let picking = $state(false);
  let owners = $state<string[]>([]);
  let owner = $state("");
  let projects = $state<ProjectSummary[] | null>(null);
  let chosen = $state(0);

  // filtros
  let text = $state("");
  let typeId = $state("");
  let onlyMine = $state(false);
  let limits = $state<Record<string, number>>({});
  let colsOpen = $state(false);

  // arrastar
  let dragging = $state<string | null>(null);
  let over = $state<string | null>(null);

  let creatingIn = $state<string | null>(null);
  /** issues abertas no painel, da primeira à atual (sub-issues e mães empilham para o Voltar) */
  let stack = $state<string[]>([]);

  const allCols = $derived(board ? columns(board, { text, typeId, onlyMine, viewer }) : []);
  const cols = $derived(allCols.filter((c) => !prefs.hidden.includes(c.id)));
  const shown = $derived(cols.reduce((n, c) => n + c.cards.length, 0));
  const openIssue = $derived(stack.at(-1) ?? null);
  const openCard = $derived(board?.cards.find((c) => c.kind === "issue" && c.contentId === openIssue) ?? null);
  const repos = $derived([...new Set([...(board?.repos ?? []), ...watchedRepos])].sort((a, b) => a.localeCompare(b)));
  const scopeHint = $derived(/scope|escopo|project/i.test(error));

  onMount(async () => {
    api.boardViewer().then((v) => (viewer = v)).catch(() => {});
    try {
      prefs = await api.boardPrefs();
    } catch (e) {
      error = errorMessage(e);
    }
    if (prefs.owner && prefs.number) void load();
    else void startPicking();
  });

  async function load(): Promise<void> {
    loading = true;
    error = "";
    try {
      board = await api.loadBoard(prefs.owner, prefs.number);
      if (typeId && !board.type?.options.some((o) => o.id === typeId)) typeId = "";
    } catch (e) {
      error = errorMessage(e);
    } finally {
      loading = false;
    }
  }

  async function startPicking(): Promise<void> {
    picking = true;
    error = "";
    chosen = prefs.number;
    try {
      if (!owners.length) owners = await api.projectOwners();
      await pickOwner(prefs.owner && owners.includes(prefs.owner) ? prefs.owner : (owners[0] ?? ""));
    } catch (e) {
      error = errorMessage(e);
    }
  }

  async function pickOwner(o: string): Promise<void> {
    owner = o;
    projects = null;
    if (!o) return;
    try {
      projects = await api.listProjects(o);
    } catch (e) {
      error = errorMessage(e);
      projects = [];
    }
  }

  async function openProject(): Promise<void> {
    if (!owner || !chosen) return;
    const same = prefs.owner === owner && prefs.number === chosen;
    prefs = { ...prefs, owner, number: chosen, hidden: same ? prefs.hidden : [] };
    picking = false;
    board = null;
    limits = {};
    void api.saveBoardPrefs(prefs).catch(() => {});
    await load();
  }

  function ago(iso: string): string {
    const t = Date.parse(iso);
    if (!t) return "";
    const min = Math.round((Date.now() - t) / 60000);
    if (min < 60) return min < 1 ? "agora" : `há ${min} min`;
    const h = Math.round(min / 60);
    if (h < 24) return `há ${h} h`;
    const d = Math.round(h / 24);
    return d < 14 ? `há ${d} d` : `há ${Math.round(d / 7)} sem`;
  }

  function toggleColumn(id: string): void {
    const hidden = prefs.hidden.includes(id) ? prefs.hidden.filter((h) => h !== id) : [...prefs.hidden, id];
    prefs = { ...prefs, hidden };
    void api.saveBoardPrefs(prefs).catch(() => {});
  }

  function patch(itemId: string, p: Partial<BoardCard>): void {
    if (!board) return;
    board = { ...board, cards: board.cards.map((c) => (c.itemId === itemId ? { ...c, ...p } : c)) };
  }

  async function move(itemId: string, statusId: string): Promise<void> {
    const b = board;
    const card = b?.cards.find((c) => c.itemId === itemId);
    if (!b || !card || card.statusId === statusId) return;
    const before = card.statusId;
    board = moveCard(b, itemId, statusId);
    try {
      await api.setCardField(b.projectId, itemId, b.status.id, statusId);
    } catch (e) {
      patch(itemId, { statusId: before });
      notice = `Não consegui mover #${card.number}: ${errorMessage(e)}`;
    }
  }

  function onDragStart(e: DragEvent, card: BoardCard): void {
    dragging = card.itemId;
    e.dataTransfer?.setData("text/plain", card.itemId);
    if (e.dataTransfer) e.dataTransfer.effectAllowed = "move";
  }

  function onDrop(e: DragEvent, colId: string): void {
    e.preventDefault();
    const id = e.dataTransfer?.getData("text/plain") || dragging;
    dragging = over = null;
    if (id) void move(id, colId);
  }

  function created(res: CreatedCard): void {
    creatingIn = null;
    if (board && res.card.itemId) board = { ...board, cards: [res.card, ...board.cards] };
    notice = res.warning || `Card #${res.card.number} criado em ${res.card.repo}.`;
    if (res.warning) void load();
  }

  function openCardOf(card: BoardCard): void {
    if (card.kind === "issue") stack = [card.contentId];
    else if (card.url) void api.openURL(card.url);
  }

  function typeOf(card: BoardCard): string {
    return board?.type?.options.find((o) => o.id === card.typeId)?.name ?? "";
  }
</script>

<div class="board">
  {#if picking}
    <section class="picker">
      <div class="intro">
        <h2>Qual projeto você quer abrir?</h2>
        <p class="muted">O Oddeye lembra a escolha. Dá para trocar depois em "Trocar projeto".</p>
      </div>

      {#if owners.length}
        <div class="block">
          <span class="label">Dono</span>
          <div class="seg owners" role="group" aria-label="Dono do projeto">
            {#each owners as o (o)}
              <button class:on={owner === o} aria-pressed={owner === o} onclick={() => pickOwner(o)}>{o}</button>
            {/each}
          </div>
        </div>
      {/if}

      <div class="plist">
        {#if projects === null}
          <p class="muted pad">{error ? "" : "Carregando projetos..."}</p>
        {:else if projects.length === 0}
          <p class="muted pad">Nenhum projeto aberto em {owner}.</p>
        {:else}
          {#each projects as p (p.id)}
            <button class="prow" class:on={chosen === p.number} onclick={() => (chosen = p.number)} ondblclick={openProject}>
              <svg class="lock" class:hide={p.public} viewBox="0 0 14 14" aria-label={p.public ? "Público" : "Privado"} role="img"
                ><rect x="2.5" y="6" width="9" height="6.5" rx="1" /><path d="M4.5 6V4.5a2.5 2.5 0 0 1 5 0V6" /></svg
              >
              <span class="ptitle">
                <span>{p.title}</span>
                <span class="muted small">{p.items} ite{p.items === 1 ? "m" : "ns"}</span>
              </span>
              <span class="mono muted small">#{p.number}</span>
              <span class="muted small when">{ago(p.updatedAt)}</span>
            </button>
          {/each}
        {/if}
      </div>

      {#if error}<p class="error-text">{error}{scopeHint ? " Autorize o escopo project pelo aviso acima e tente de novo." : ""}</p>{/if}

      <div class="actions">
        {#if board}<button class="btn" onclick={() => (picking = false)}>Cancelar</button>{/if}
        <button class="btn primary" disabled={!chosen || !projects?.some((p) => p.number === chosen)} onclick={openProject}>Abrir board</button>
      </div>
    </section>
  {:else}
    <div class="bhead">
      {#if board && !board.public}
        <svg class="lock" viewBox="0 0 14 14" aria-label="Privado" role="img"><rect x="2.5" y="6" width="9" height="6.5" rx="1" /><path d="M4.5 6V4.5a2.5 2.5 0 0 1 5 0V6" /></svg>
      {/if}
      <h2>{board?.title ?? (loading ? "Carregando o board..." : "Board")}</h2>
      <span class="mono muted small">{prefs.owner} · projeto #{prefs.number}</span>
      <button class="link quiet" onclick={startPicking}>Trocar projeto</button>
      <span class="grow"></span>
      {#if board}<button class="btn" onclick={() => api.openURL(board!.url)}>Abrir no GitHub</button>{/if}
      <button class="btn" onclick={load} disabled={loading}>{loading ? "Atualizando..." : "Atualizar"}</button>
      <button class="btn primary" disabled={!board} onclick={() => (creatingIn = board?.status.options[0]?.id ?? "")}>
        <svg viewBox="0 0 14 14" aria-hidden="true"><path d="M7 2.5v9M2.5 7h9" /></svg>
        Novo card
      </button>
    </div>

    {#if error}<p class="error-text">{error}{scopeHint ? " Autorize o escopo project pelo aviso acima e tente de novo." : ""}</p>{/if}

    {#if board}
      <div class="filters">
        <label class="search">
          <svg viewBox="0 0 14 14" aria-hidden="true"><circle cx="6" cy="6" r="4" /><path d="M9 9l3 3" /></svg>
          <input class="field" bind:value={text} placeholder="Buscar por título, número ou repositório" aria-label="Buscar cards" />
        </label>
        {#if board.type}
          <div class="seg" role="group" aria-label={board.type.name}>
            <button class:on={typeId === ""} onclick={() => (typeId = "")}>Todos</button>
            {#each board.type.options as o (o.id)}
              <button class:on={typeId === o.id} aria-pressed={typeId === o.id} onclick={() => (typeId = o.id)}>{o.name}</button>
            {/each}
          </div>
        {/if}
        {#if viewer}
          <div class="seg" role="group" aria-label="Responsável">
            <button class:on={!onlyMine} onclick={() => (onlyMine = false)}>Todos</button>
            <button class:on={onlyMine} aria-pressed={onlyMine} onclick={() => (onlyMine = true)}>Só meus</button>
          </div>
        {/if}
        <button class="btn" class:on={colsOpen} aria-expanded={colsOpen} onclick={() => (colsOpen = !colsOpen)}>
          Colunas{prefs.hidden.length ? ` (${prefs.hidden.length} oculta${prefs.hidden.length === 1 ? "" : "s"})` : ""}
        </button>
        <span class="grow"></span>
        <span class="muted">{shown} de {board.cards.length} cards{board.truncated ? " (board truncado em 1000)" : ""}</span>
      </div>

      {#if colsOpen}
        <div class="colpick" role="group" aria-label="Colunas visíveis">
          {#each allCols as c (c.id)}
            <label><input type="checkbox" checked={!prefs.hidden.includes(c.id)} onchange={() => toggleColumn(c.id)} /> {c.name} <span class="muted small">{c.cards.length}</span></label>
          {/each}
        </div>
      {/if}

      <div class="cols">
        {#each cols as col (col.id)}
          {@const tone = columnTone(col.name)}
          {@const limit = limits[col.id] ?? PAGE}
          <section
            class="col"
            class:over={over === col.id && dragging !== null}
            aria-label={col.name}
            ondragover={(e) => {
              e.preventDefault();
              over = col.id;
            }}
            ondragleave={(e) => {
              if (!(e.currentTarget as HTMLElement).contains(e.relatedTarget as Node)) over = null;
            }}
            ondrop={(e) => onDrop(e, col.id)}
          >
            <div class="colhead">
              <span class="dot {tone}"></span>
              <h3>{col.name}</h3>
              <span class="count">{col.cards.length}</span>
              <button class="plus" aria-label="Novo card em {col.name}" title="Novo card em {col.name}" onclick={() => (creatingIn = col.id)}>
                <svg viewBox="0 0 14 14" aria-hidden="true"><path d="M7 2.5v9M2.5 7h9" /></svg>
              </button>
            </div>
            {#if col.description}<p class="cdesc">{col.description}</p>{/if}

            {#each col.cards.slice(0, limit) as c (c.itemId)}
              {@const t = typeOf(c)}
              <button
                class="card"
                class:dragging={dragging === c.itemId}
                draggable="true"
                ondragstart={(e) => onDragStart(e, c)}
                ondragend={() => (dragging = over = null)}
                onclick={() => openCardOf(c)}
              >
                <span class="cline">
                  <span class="mono muted small">
                    {#if c.kind === "draft"}rascunho{:else}{c.repo.split("/")[1]} #{c.number}{/if}
                    {#if c.kind === "pr"} · PR{/if}
                  </span>
                  <span class="grow"></span>
                  {#each c.assignees.slice(0, 2) as a (a)}
                    <span class="av" title={a}>{initials(a)}</span>
                  {/each}
                </span>
                <span class="ctitle">{c.title}</span>
                {#if t || c.subTotal}
                  <span class="cline">
                    {#if t}<span class="chip" class:bug={/bug/i.test(t)}>{t}</span>{/if}
                    {#if c.subTotal}
                      <span class="bar"><span class:full={c.subDone === c.subTotal} style="width: {(c.subDone / c.subTotal) * 100}%"></span></span>
                      <span class="mono muted small">{c.subDone}/{c.subTotal}</span>
                    {/if}
                  </span>
                {/if}
              </button>
            {/each}
            {#if col.cards.length > limit}
              <button class="link quiet more" onclick={() => (limits = { ...limits, [col.id]: limit + PAGE })}>
                Mostrar mais {Math.min(PAGE, col.cards.length - limit)} de {col.cards.length - limit}
              </button>
            {/if}
          </section>
        {/each}
      </div>
    {/if}
  {/if}
</div>

{#if notice}
  <div class="notice" role="status">
    <span>{notice}</span>
    <button class="link" onclick={() => (notice = "")}>Fechar</button>
  </div>
{/if}

{#if board && creatingIn !== null}
  <NewCardModal {board} statusId={creatingIn} {repos} lastRepo={prefs.lastRepo} {viewer} oncreated={created} onclose={() => (creatingIn = null)} />
{/if}

{#if board && openIssue}
  {#key openIssue}
    <CardPanel
      {board}
      issueId={openIssue}
      card={openCard}
      {viewer}
      onpatch={patch}
      onopen={(id) => (stack = [...stack, id])}
      onback={stack.length > 1 ? () => (stack = stack.slice(0, -1)) : undefined}
      onclose={() => (stack = [])}
    />
  {/key}
{/if}

<style>
  .board {
    display: flex;
    flex-direction: column;
    gap: var(--sp-4);
    min-width: 0;
  }
  /* cabeçalho, filtros e escolha do projeto numa coluna central um pouco mais larga que a da home,
     para a barra do board caber numa linha; a faixa de colunas usa a largura toda, desde a esquerda */
  .board > :not(.cols) {
    width: min(1280px, 100%);
    margin-inline: auto;
    box-sizing: border-box;
  }
  .grow {
    flex: 1;
  }
  .muted {
    color: var(--muted);
  }
  .small {
    font-size: var(--fs-xs);
  }
  h2 {
    margin: 0;
    font-size: var(--fs-lg);
    font-weight: 600;
  }
  .lock {
    width: 16px;
    height: 16px;
    flex-shrink: 0;
    fill: none;
    stroke: var(--muted);
    stroke-width: 1.5;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .lock.hide {
    visibility: hidden;
  }

  /* escolha do projeto */
  .picker {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }
  .picker > * {
    max-width: 720px;
  }
  .intro p {
    margin: 4px 0 0;
  }
  .block {
    display: flex;
    flex-direction: column;
    gap: var(--sp-2);
  }
  .owners {
    align-self: flex-start;
    flex-wrap: wrap;
  }
  .plist {
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    background: var(--surface);
    overflow: hidden;
  }
  .pad {
    margin: 0;
    padding: var(--sp-4);
  }
  .prow {
    display: flex;
    align-items: center;
    gap: var(--sp-3);
    width: 100%;
    padding: var(--sp-3) var(--sp-4);
    border: none;
    border-top: 1px solid var(--border);
    background: transparent;
    color: var(--text);
    font: inherit;
    text-align: left;
    transition: background var(--ease);
  }
  .prow:first-child {
    border-top: none;
  }
  .prow:hover {
    background: var(--surface-2);
  }
  .prow.on {
    background: var(--selected);
  }
  .ptitle {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
    font-weight: 500;
  }
  .when {
    width: 72px;
    text-align: right;
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--sp-2);
  }

  /* board */
  .bhead,
  .filters {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--sp-3);
  }
  .search {
    position: relative;
    display: flex;
    align-items: center;
    flex: 1 1 280px;
    max-width: 420px;
  }
  .search svg {
    position: absolute;
    left: 12px;
    width: 14px;
    height: 14px;
    fill: none;
    stroke: var(--muted);
    stroke-width: 1.5;
    stroke-linecap: round;
  }
  .search .field {
    padding-left: 34px;
  }
  .btn.on {
    background: var(--selected);
  }
  .colpick {
    display: flex;
    flex-wrap: wrap;
    gap: var(--sp-2) var(--sp-5);
    padding: var(--sp-3) var(--sp-4);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    background: var(--surface);
  }
  .colpick label {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
    cursor: pointer;
  }
  .cols {
    display: flex;
    align-items: flex-start;
    gap: var(--sp-3);
    overflow-x: auto;
    padding-bottom: var(--sp-3);
  }
  .col {
    display: flex;
    flex-direction: column;
    gap: var(--sp-2);
    flex: 0 0 300px;
    padding: var(--sp-3);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    background: var(--surface);
    transition: border-color var(--ease);
  }
  .col.over {
    border-color: var(--muted);
    border-style: dashed;
  }
  .colhead {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
  }
  .colhead h3 {
    margin: 0;
    font-size: var(--fs-sm);
    font-weight: 600;
  }
  .dot {
    width: 10px;
    height: 10px;
    flex-shrink: 0;
    border: 1.5px solid var(--muted);
    border-radius: 50%;
  }
  .dot.blocked {
    border-color: var(--red);
    background: var(--red);
  }
  .dot.done {
    border-color: var(--green);
    background: var(--green);
  }
  .count {
    padding: 0 7px;
    border-radius: 999px;
    background: var(--selected);
    color: var(--muted);
    font-size: var(--fs-xs);
  }
  .plus {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    margin-left: auto;
    border: none;
    border-radius: var(--r-sm);
    background: transparent;
    color: var(--muted);
    transition: background var(--ease), color var(--ease);
  }
  .plus:hover {
    background: var(--selected);
    color: var(--text);
  }
  .plus svg,
  .bhead svg {
    width: 14px;
    height: 14px;
    fill: none;
    stroke: currentColor;
    stroke-width: 1.5;
    stroke-linecap: round;
  }
  .cdesc {
    margin: 0 0 4px;
    color: var(--muted);
    font-size: var(--fs-xs);
  }
  .card {
    display: flex;
    flex-direction: column;
    gap: var(--sp-2);
    width: 100%;
    padding: var(--sp-3);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    background: var(--surface-2);
    color: var(--text);
    font: inherit;
    text-align: left;
    cursor: grab;
    transition: border-color var(--ease), opacity var(--ease);
  }
  .card:hover {
    border-color: var(--border-strong);
  }
  .card.dragging {
    opacity: 0.4;
  }
  .cline {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
    width: 100%;
  }
  .ctitle {
    font-size: var(--fs-md);
    line-height: 1.4;
    overflow-wrap: anywhere;
  }
  .av {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    flex-shrink: 0;
    border-radius: 50%;
    background: var(--border-strong);
    font-size: 10px;
    font-weight: 600;
  }
  .chip {
    display: inline-flex;
    align-items: center;
    height: 20px;
    padding: 0 var(--sp-2);
    border: 1px solid var(--border-strong);
    border-radius: 999px;
    color: var(--text-2);
    font-size: var(--fs-xs);
    white-space: nowrap;
  }
  .chip.bug {
    border-color: rgba(248, 81, 73, 0.45);
    background: var(--red-bg);
    color: var(--red);
  }
  .bar {
    flex: 1;
    height: 4px;
    border-radius: 2px;
    background: var(--border-strong);
    overflow: hidden;
  }
  .bar span {
    display: block;
    height: 100%;
    background: var(--text);
  }
  .bar span.full {
    background: var(--green);
  }
  .more {
    align-self: center;
    font-size: var(--fs-xs);
  }
  .notice {
    position: fixed;
    left: 50%;
    bottom: var(--sp-5);
    z-index: var(--z-toast);
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    gap: var(--sp-4);
    max-width: min(640px, calc(100% - 32px));
    padding: var(--sp-3) var(--sp-4);
    border: 1px solid var(--border-strong);
    border-radius: var(--r-md);
    background: var(--surface);
    box-shadow: var(--shadow);
  }
</style>
