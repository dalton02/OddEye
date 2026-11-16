<script lang="ts">
  import { onMount } from "svelte";
  import { api, errorMessage, type IssueDetail } from "./api";
  import { branchName, type Board, type BoardCard } from "./board";

  interface Props {
    board: Board;
    /** id do nó da issue mostrada */
    issueId: string;
    /** o card dela no board; null quando a issue não está no projeto (ou não foi carregada nele) */
    card: BoardCard | null;
    viewer: string;
    /** aplica uma mudança no card (o board atualiza na hora; reverte se a chamada falhar) */
    onpatch: (itemId: string, patch: Partial<BoardCard>) => void;
    /** abre outra issue (sub-issue ou mãe) no painel */
    onopen: (issueId: string) => void;
    /** volta para a issue anterior; ausente quando não há para onde voltar */
    onback?: () => void;
    onclose: () => void;
  }

  let { board, issueId, card, viewer, onpatch, onopen, onback, onclose }: Props = $props();

  let detail = $state<IssueDetail | null>(null);
  let loading = $state(true);
  let error = $state("");
  let busy = $state("");

  let newSub = $state("");
  let branch = $state("");
  let base = $state("");
  let copied = $state("");

  const typeName = $derived(board.type?.options.find((o) => o.id === card?.typeId)?.name ?? "");
  const number = $derived(card?.number ?? detail?.number ?? 0);
  const repo = $derived(card?.repo ?? detail?.repo ?? "");
  const url = $derived(card?.url ?? detail?.url ?? "");
  const assignees = $derived(card?.assignees ?? detail?.assignees ?? []);
  const mine = $derived(assignees.some((a) => a.toLowerCase() === viewer.toLowerCase()));
  // status das sub-issues novas: a primeira opção que parece "a fazer"
  const todoOption = $derived(board.status.options.find((o) => /todo|to do|a fazer|backlog/i.test(o.name)));

  onMount(() => {
    load();
  });

  async function load(): Promise<void> {
    loading = true;
    error = "";
    try {
      detail = await api.issueDetail(issueId);
      branch = branchName(typeName, detail.number, detail.title);
      base = detail.defaultBranch;
    } catch (e) {
      error = errorMessage(e);
    } finally {
      loading = false;
    }
  }

  async function run(label: string, fn: () => Promise<void>): Promise<void> {
    busy = label;
    error = "";
    try {
      await fn();
    } catch (e) {
      error = errorMessage(e);
    } finally {
      busy = "";
    }
  }

  function setField(card: BoardCard, fieldId: string, key: "statusId" | "typeId", optionId: string): void {
    const before = card[key];
    onpatch(card.itemId, { [key]: optionId });
    void run("field", async () => {
      try {
        await api.setCardField(board.projectId, card.itemId, fieldId, optionId);
      } catch (e) {
        onpatch(card.itemId, { [key]: before });
        throw e;
      }
    });
  }

  function toggleMine(): void {
    const assign = !mine;
    void run("assign", async () => {
      await api.setAssignedToMe(issueId, assign);
      const rest = assignees.filter((a) => a.toLowerCase() !== viewer.toLowerCase());
      const next = assign ? [viewer, ...rest] : rest;
      if (detail) detail.assignees = next;
      if (card) onpatch(card.itemId, { assignees: next });
    });
  }

  function countSubs(d: IssueDetail): void {
    if (card) onpatch(card.itemId, { subTotal: d.subIssues.length, subDone: d.subIssues.filter((s) => s.state === "CLOSED").length });
  }

  function toggleSub(id: string): void {
    const d = detail;
    const sub = d?.subIssues.find((s) => s.id === id);
    if (!d || !sub) return;
    const closed = sub.state !== "CLOSED";
    void run("sub:" + id, async () => {
      await api.setIssueClosed(id, closed);
      sub.state = closed ? "CLOSED" : "OPEN";
      countSubs(d);
    });
  }

  function addSub(e: SubmitEvent): void {
    e.preventDefault();
    const d = detail;
    const title = newSub.trim();
    if (!d || !title || busy) return;
    void run("newsub", async () => {
      const sub = await api.createSubIssue(board.projectId, d.id, d.repo, title, board.status.id, todoOption?.id ?? "");
      d.subIssues.push(sub);
      newSub = "";
      countSubs(d);
    });
  }

  function createBranch(): void {
    const d = detail;
    const ref = d?.branches.find((b) => b.name === base);
    if (!d || !ref || !branch.trim()) return;
    void run("branch", async () => {
      const name = await api.createIssueBranch(d.id, d.repoId, branch.trim(), ref.oid);
      d.linked.push(name);
    });
  }

  async function copyCheckout(name: string): Promise<void> {
    try {
      await navigator.clipboard.writeText(`git fetch origin && git checkout ${name}`);
      copied = name;
      setTimeout(() => copied === name && (copied = ""), 1500);
    } catch {
      error = "Não consegui copiar para a área de transferência.";
    }
  }

  function onKey(e: KeyboardEvent): void {
    if (e.key !== "Escape") return;
    if (onback) onback();
    else onclose();
  }
</script>

<svelte:window onkeydown={onKey} />

<div class="scrim" role="presentation" onclick={onclose}></div>
<aside class="panel" aria-label="Issue {repo} #{number}">
  <div class="head">
    {#if onback}
      <button class="btn small square" aria-label="Voltar" title="Voltar (Esc)" onclick={onback}>
        <svg viewBox="0 0 14 14" aria-hidden="true"><path d="M8.5 3L4.5 7l4 4" /></svg>
      </button>
    {/if}
    <span class="mono ref">{repo}{number ? ` #${number}` : ""}</span>
    <span class="grow"></span>
    <button class="btn small" disabled={!url} onclick={() => api.openURL(url)}>Abrir no GitHub</button>
    <button class="btn small square" aria-label="Fechar" onclick={onclose}>
      <svg viewBox="0 0 14 14" aria-hidden="true"><path d="M3 3l8 8M11 3l-8 8" /></svg>
    </button>
  </div>

  <div class="body">
    {#if detail?.parent}
      {@const p = detail.parent}
      <button class="parent" onclick={() => onopen(p.id)} title="Abrir a issue mãe">
        <svg viewBox="0 0 14 14" aria-hidden="true"><path d="M4 11V5.5A2.5 2.5 0 0 1 6.5 3H11M8.5 1l2.5 2-2.5 2" /></svg>
        Sub-issue de <span class="mono">#{p.number}</span> <span class="ptitle">{p.title}</span>
      </button>
    {/if}
    <h2>{detail?.title ?? card?.title ?? ""}</h2>

    <div class="fields">
      {#if card}
      {@const c = card}
      <label>
        <span class="label">Status</span>
        <select class="field" value={c.statusId} onchange={(e) => setField(c, board.status.id, "statusId", e.currentTarget.value)}>
          {#if !c.statusId}<option value="">Sem status</option>{/if}
          {#each board.status.options as o (o.id)}<option value={o.id}>{o.name}</option>{/each}
        </select>
      </label>
      {#if board.type}
        {@const tf = board.type}
        <label>
          <span class="label">{tf.name}</span>
          <select class="field" value={c.typeId} onchange={(e) => setField(c, tf.id, "typeId", e.currentTarget.value)}>
            <option value="">Sem tipo</option>
            {#each tf.options as o (o.id)}<option value={o.id}>{o.name}</option>{/each}
          </select>
        </label>
      {/if}
      {:else if detail}
      <div class="assign">
        <span class="label">Estado</span>
        <span class="who">{detail.state === "CLOSED" ? "Fechada" : "Aberta"} · fora do board</span>
      </div>
      {/if}
      <div class="assign">
        <span class="label">Responsável</span>
        <span class="who">{assignees.length ? assignees.join(", ") : "Ninguém"}</span>
        {#if viewer}
          <button class="link quiet" disabled={busy === "assign"} onclick={toggleMine}>{mine ? "Tirar de mim" : "Atribuir a mim"}</button>
        {/if}
      </div>
    </div>

    {#if error}<p class="error-text">{error}</p>{/if}

    {#if loading}
      <p class="muted">Carregando a issue...</p>
    {:else if detail}
      {@const d = detail}
      <section>
        <h3>Descrição</h3>
        {#if d.body.trim()}
          <p class="desc">{d.body}</p>
        {:else}
          <p class="muted">Sem descrição.</p>
        {/if}
      </section>

      <section>
        <div class="sech">
          <h3>Sub-issues</h3>
          {#if d.subIssues.length}
            {@const done = d.subIssues.filter((s) => s.state === "CLOSED").length}
            <span class="mono muted">{done}/{d.subIssues.length}</span>
            <span class="bar"><span class:full={done === d.subIssues.length} style="width: {(done / d.subIssues.length) * 100}%"></span></span>
          {/if}
        </div>
        {#if d.subIssues.length}
          <ul class="subs">
            {#each d.subIssues as s (s.id)}
              <li>
                <input
                  type="checkbox"
                  checked={s.state === "CLOSED"}
                  disabled={busy === "sub:" + s.id}
                  aria-label={s.state === "CLOSED" ? `Reabrir #${s.number}` : `Fechar #${s.number}`}
                  onchange={() => toggleSub(s.id)}
                />
                <button class="sub-open" onclick={() => onopen(s.id)} title="Abrir #{s.number} aqui">
                  <span class="mono muted">#{s.number}</span>
                  <span class:closed={s.state === "CLOSED"}>{s.title}</span>
                </button>
                <button class="ext" aria-label="Abrir #{s.number} no GitHub" title="Abrir no GitHub" onclick={() => api.openURL(s.url)}>
                  <svg viewBox="0 0 14 14" aria-hidden="true"><path d="M8 2.5h3.5V6M11.5 2.5L6 8M10 8.5v3H2.5V4h3" /></svg>
                </button>
              </li>
            {/each}
          </ul>
        {/if}
        <form class="newsub" onsubmit={addSub}>
          <svg viewBox="0 0 14 14" aria-hidden="true"><path d="M7 2.5v9M2.5 7h9" /></svg>
          <input bind:value={newSub} placeholder="Nova sub-issue: escreva o título e aperte Enter" aria-label="Nova sub-issue" disabled={busy === "newsub"} />
        </form>
        <span class="hint">
          {busy === "newsub" ? "Criando..." : `Nasce em ${d.repo}, ligada a #${d.number}${todoOption ? `, no projeto com status ${todoOption.name}` : ", no projeto"}.`}
        </span>
      </section>

      <section>
        <h3>Branch</h3>
        <div class="branch">
          <label>
            <span class="label">Nome</span>
            <input class="field mono" bind:value={branch} spellcheck="false" />
          </label>
          <label>
            <span class="label">A partir de</span>
            <select class="field mono" bind:value={base}>
              {#each d.branches as b (b.name)}<option value={b.name}>{b.name}</option>{/each}
            </select>
          </label>
        </div>
        <div class="row">
          <button class="btn primary" disabled={!branch.trim() || !base || busy === "branch"} onclick={createBranch}>
            {busy === "branch" ? "Criando..." : "Criar branch remota"}
          </button>
          <span class="hint">Fica ligada à issue em "Development" no GitHub.</span>
        </div>
        {#if d.linked.length}
          <div class="linked">
            <span class="label">Branches ligadas</span>
            {#each d.linked as name (name)}
              <div class="row">
                <span class="mono grow">{name}</span>
                <button class="btn small" onclick={() => copyCheckout(name)}>{copied === name ? "Copiado" : "Copiar git checkout"}</button>
              </div>
            {/each}
          </div>
        {/if}
      </section>
    {/if}
  </div>
</aside>

<style>
  .scrim {
    position: fixed;
    inset: 0;
    z-index: var(--z-drawer);
    background: rgba(0, 0, 0, 0.5);
  }
  .panel {
    position: fixed;
    top: 0;
    right: 0;
    bottom: 0;
    z-index: var(--z-drawer);
    width: min(560px, 100%);
    display: flex;
    flex-direction: column;
    border-left: 1px solid var(--border-strong);
    background: var(--surface);
    box-shadow: var(--shadow);
  }
  .head {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
    padding: var(--sp-3) 20px;
    border-bottom: 1px solid var(--border);
  }
  .ref {
    color: var(--muted);
    font-size: var(--fs-xs);
  }
  .grow {
    flex: 1;
    min-width: 0;
  }
  .body {
    flex: 1;
    overflow-y: auto;
    padding: 20px;
  }
  h2 {
    margin: 0 0 var(--sp-4);
    font-size: var(--fs-lg);
    font-weight: 600;
    line-height: 1.3;
  }
  h3 {
    margin: 0;
    font-size: var(--fs-sm);
    font-weight: 600;
  }
  .fields {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: var(--sp-3);
    padding-bottom: var(--sp-4);
  }
  .fields label,
  .branch label,
  .assign {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .assign .who {
    min-height: 20px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .assign .link {
    align-self: flex-start;
    font-size: var(--fs-xs);
  }
  section {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: var(--sp-4) 0;
    border-top: 1px solid var(--border);
  }
  .sech,
  .row {
    display: flex;
    align-items: center;
    gap: var(--sp-3);
  }
  .muted,
  .hint {
    margin: 0;
    color: var(--muted);
  }
  .hint {
    font-size: var(--fs-xs);
  }
  .desc {
    margin: 0;
    max-height: 220px;
    overflow: auto;
    color: var(--text-2);
    white-space: pre-wrap;
    overflow-wrap: anywhere;
  }
  .bar {
    flex: 0 1 160px;
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
  .subs {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .subs li {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px 8px;
    border-radius: var(--r-sm);
  }
  .subs li:hover {
    background: var(--surface-2);
  }
  .sub-open {
    display: flex;
    gap: var(--sp-2);
    flex: 1;
    min-width: 0;
    padding: 0;
    border: none;
    background: none;
    color: var(--text);
    font: inherit;
    text-align: left;
  }
  .head svg,
  .parent svg,
  .ext svg {
    width: 14px;
    height: 14px;
    fill: none;
    stroke: currentColor;
    stroke-width: 1.5;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .parent {
    display: flex;
    align-items: center;
    gap: 6px;
    max-width: 100%;
    margin-bottom: var(--sp-2);
    padding: 0;
    border: none;
    background: none;
    color: var(--muted);
    font: inherit;
    font-size: var(--fs-xs);
    text-align: left;
    transition: color var(--ease);
  }
  .parent:hover {
    color: var(--text);
  }
  .parent .ptitle {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .sub-open:hover span:last-child {
    text-decoration: underline;
    text-decoration-color: var(--muted);
    text-underline-offset: 3px;
  }
  .ext {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    flex-shrink: 0;
    padding: 0;
    border: none;
    border-radius: var(--r-sm);
    background: transparent;
    color: var(--muted);
    opacity: 0;
    transition: opacity var(--ease), color var(--ease);
  }
  .subs li:hover .ext,
  .ext:focus-visible {
    opacity: 1;
  }
  .ext:hover {
    color: var(--text);
  }
  .closed {
    color: var(--muted);
    text-decoration: line-through;
  }
  .newsub {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
    padding: 0 10px;
    border: 1px dashed var(--border-strong);
    border-radius: var(--r-sm);
  }
  .newsub:focus-within {
    border-color: var(--primary);
    border-style: solid;
  }
  .newsub svg {
    width: 14px;
    height: 14px;
    flex-shrink: 0;
    fill: none;
    stroke: var(--muted);
    stroke-width: 1.5;
    stroke-linecap: round;
  }
  .newsub input {
    flex: 1;
    height: 36px;
    border: none;
    outline: none;
    background: transparent;
    color: var(--text);
    font: inherit;
  }
  .branch {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 160px;
    gap: var(--sp-3);
  }
  .branch .field {
    font-size: var(--fs-xs);
  }
  .linked {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 10px var(--sp-3);
    border: 1px solid var(--border);
    border-radius: var(--r-sm);
    background: var(--surface-2);
  }
  .linked .mono {
    font-size: var(--fs-xs);
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
