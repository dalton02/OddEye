<script lang="ts">
  import { onMount } from "svelte";
  import { api, errorMessage, type Config, type RepoConfig, type RepoOption } from "./api";

  interface Props {
    onclose: () => void;
  }

  let { onclose }: Props = $props();

  let config = $state<Config | null>(null);
  let defaultPrompt = $state("");
  let configDir = $state("");
  let prompt = $state("");
  let defaultArchitectPrompt = $state("");
  let architectPrompt = $state("");
  let error = $state("");
  let saving = $state(false);
  let saved = $state(false);

  let repoOptions = $state<RepoOption[] | null>(null);
  let repoError = $state("");
  let query = $state("");
  let expanded = $state<string | null>(null);

  const intervals = [2, 5, 10, 15, 30, 60];

  type Tab = "repos" | "notifications" | "reviewer" | "about";
  const tabs: { id: Tab; label: string }[] = [
    { id: "repos", label: "Repositórios" },
    { id: "notifications", label: "Notificações" },
    { id: "reviewer", label: "Revisor" },
    { id: "about", label: "Sobre" }
  ];
  let tab = $state<Tab>("repos");
  /* as instruções são longas: ficam fechadas até a pessoa pedir para editar */
  let editingPrompt = $state(false);
  let editingArchitect = $state(false);

  let version = $state("");
  let updateStatus = $state("");
  let checkingUpdate = $state(false);

  /* se houver versão nova, o backend emite update:available e o App abre o modal */
  async function checkUpdate(): Promise<void> {
    checkingUpdate = true;
    updateStatus = "";
    try {
      const u = await api.checkUpdate();
      updateStatus = u ? `Versão ${u.version} disponível.` : "Você já está na versão mais recente.";
    } catch (e) {
      updateStatus = errorMessage(e);
    } finally {
      checkingUpdate = false;
    }
  }

  onMount(async () => {
    api.version().then((v) => (version = v));
    try {
      const s = await api.getSettings();
      config = s.config;
      defaultPrompt = s.defaultPrompt;
      configDir = s.configDir;
      prompt = s.config.reviewerPrompt || s.defaultPrompt;
      defaultArchitectPrompt = s.defaultArchitectPrompt;
      architectPrompt = s.config.architectPrompt || s.defaultArchitectPrompt;
    } catch (e) {
      error = errorMessage(e);
    }
    api
      .listRepos()
      .then((r) => (repoOptions = r))
      .catch((e) => (repoError = errorMessage(e)));
  });

  const key = (r: { owner: string; name: string }): string => `${r.owner}/${r.name}`.toLowerCase();

  const suggestions = $derived.by(() => {
    if (!config || !repoOptions) return [];
    const added = new Set(config.repos.map(key));
    const q = query.trim().toLowerCase();
    return repoOptions
      .filter((r) => !added.has(r.fullName.toLowerCase()) && (!q || r.fullName.toLowerCase().includes(q)))
      .slice(0, 8);
  });

  const typedRepo = $derived.by(() => {
    const m = query.trim().match(/^(?:https?:\/\/github\.com\/)?([\w.-]+)\/([\w.-]+?)(?:\.git)?\/?$/);
    return m ? { owner: m[1], name: m[2] } : null;
  });

  function addRepo(fullName: string): void {
    if (!config) return;
    const [owner, name] = fullName.split("/");
    if (!owner || !name || config.repos.some((r) => key(r) === `${owner}/${name}`.toLowerCase())) return;
    config.repos = [
      ...config.repos,
      { owner, name, onlyReviewRequested: false, preAnalyze: false, promptOverride: "" }
    ];
    query = "";
  }

  function removeRepo(r: RepoConfig): void {
    if (!config) return;
    config.repos = config.repos.filter((x) => key(x) !== key(r));
  }

  function onSearchKey(e: KeyboardEvent): void {
    if (e.key !== "Enter") return;
    e.preventDefault();
    if (suggestions.length > 0) addRepo(suggestions[0].fullName);
    else if (typedRepo) addRepo(`${typedRepo.owner}/${typedRepo.name}`);
  }

  async function save(): Promise<void> {
    if (!config) return;
    saving = true;
    error = "";
    try {
      const toSave: Config = {
        ...config,
        reviewerPrompt: prompt.trim() === defaultPrompt.trim() ? "" : prompt,
        architectPrompt: architectPrompt.trim() === defaultArchitectPrompt.trim() ? "" : architectPrompt
      };
      config = await api.saveSettings(toSave);
      saved = true;
      setTimeout(onclose, 600);
    } catch (e) {
      error = errorMessage(e);
    } finally {
      saving = false;
    }
  }

  function onKey(e: KeyboardEvent): void {
    if (e.key === "Escape" && !saving) onclose();
  }
</script>

<svelte:window onkeydown={onKey} />

{#snippet chevron()}
  <svg aria-hidden="true" width="14" height="14" viewBox="0 0 14 14"><path d="M3.5 5.5l3.5 3.5 3.5-3.5" /></svg>
{/snippet}

<div class="backdrop" role="presentation" onclick={(e) => e.target === e.currentTarget && !saving && onclose()}>
  <div class="modal" role="dialog" aria-modal="true" aria-labelledby="settings-title">
    <header>
      <h2 id="settings-title">Configurações</h2>
      <button class="close" onclick={onclose} aria-label="Fechar">
        <svg aria-hidden="true" width="14" height="14" viewBox="0 0 14 14"><path d="M3 3l8 8M11 3l-8 8" /></svg>
      </button>
    </header>

    {#if !config}
      <p class="loading" class:error-text={!!error}>{error || "Carregando..."}</p>
    {:else}
      <div class="layout">
        <nav aria-label="Seções">
          {#each tabs as t (t.id)}
            <button class:on={tab === t.id} aria-current={tab === t.id ? "page" : undefined} onclick={() => (tab = t.id)}>
              {t.label}
            </button>
          {/each}
        </nav>

        <div class="pane">
          {#if tab === "repos"}
            <div class="intro">
              <h3>Repositórios vigiados</h3>
              <p class="muted">O Oddeye confere esses repositórios de tempos em tempos e avisa quando chega PR nova.</p>
            </div>

            <div class="picker">
              <input
                class="field"
                bind:value={query}
                onkeydown={onSearchKey}
                placeholder={repoOptions ? "Buscar repositório ou digitar dono/repo" : "Carregando seus repositórios..."}
                aria-label="Buscar repositório"
                spellcheck="false"
                autocomplete="off"
              />
              {#if query.trim() && (suggestions.length || typedRepo)}
                <ul class="suggestions">
                  {#each suggestions as s, i (s.fullName)}
                    <li>
                      <button class:first={i === 0} onclick={() => addRepo(s.fullName)}>
                        {s.fullName}
                        {#if s.private}<span class="tag">privado</span>{/if}
                      </button>
                    </li>
                  {/each}
                  {#if suggestions.length === 0 && typedRepo}
                    <li>
                      <button class="first" onclick={() => typedRepo && addRepo(`${typedRepo.owner}/${typedRepo.name}`)}>
                        Adicionar {typedRepo.owner}/{typedRepo.name}
                      </button>
                    </li>
                  {/if}
                </ul>
              {/if}
            </div>
            {#if repoError}<p class="warn">Não consegui listar seus repositórios ({repoError}). Digite dono/repo.</p>{/if}

            {#if config.repos.length === 0}
              <p class="muted">Nenhum repositório ainda.</p>
            {:else}
              <ul class="repos">
                {#each config.repos as r (key(r))}
                  <li>
                    <div class="repo-head">
                      <span class="name mono">{r.owner}/{r.name}</span>
                      <button class="link" onclick={() => (expanded = expanded === key(r) ? null : key(r))}>
                        {expanded === key(r) ? "Fechar opções" : "Opções"}
                      </button>
                      <button class="link quiet" onclick={() => removeRepo(r)}>Remover</button>
                    </div>
                    {#if expanded === key(r)}
                      <div class="repo-opts">
                        <label class="check">
                          <input type="checkbox" bind:checked={r.onlyReviewRequested} />
                          <span>Só avisar quando pedirem o meu review</span>
                        </label>
                        <label class="check">
                          <input type="checkbox" bind:checked={r.preAnalyze} />
                          <span>
                            Pré-analisar com o Claude ao chegar
                            <small>Abrir a PR fica instantâneo, mas cada PR nova gasta uma análise da sua assinatura.</small>
                          </span>
                        </label>
                        <label class="stack">
                          <span class="label">Instruções próprias para este repositório (vazio = usa as globais)</span>
                          <textarea
                            class="field"
                            rows="3"
                            bind:value={r.promptOverride}
                            placeholder="Ex.: este repo é um SDK público; priorize compatibilidade de API."
                          ></textarea>
                        </label>
                      </div>
                    {/if}
                  </li>
                {/each}
              </ul>
            {/if}
          {:else if tab === "notifications"}
            <h3>Notificações</h3>
            <label class="stack narrow">
              <span class="label">Verificar a cada</span>
              <span class="select">
                <select class="field" bind:value={config.pollIntervalMinutes}>
                  {#each intervals as m}<option value={m}>{m} minutos</option>{/each}
                </select>
                {@render chevron()}
              </span>
            </label>
            <div class="checks">
              <label class="check"><input type="checkbox" bind:checked={config.includeDrafts} /><span>Avisar também de PRs em rascunho</span></label>
              <label class="check"><input type="checkbox" bind:checked={config.notifyOwnPRs} /><span>Avisar das minhas próprias PRs</span></label>
              <label class="check">
                <input type="checkbox" bind:checked={config.launchAtLogin} /><span>Abrir o Oddeye ao entrar no sistema (direto na bandeja)</span>
              </label>
            </div>
          {:else if tab === "reviewer"}
            <h3>Revisor (Claude)</h3>
            <label class="stack">
              <span class="label">Modelo (vazio = padrão do seu Claude Code)</span>
              <input class="field" list="models" bind:value={config.model} placeholder="padrão" spellcheck="false" />
              <datalist id="models"><option value="sonnet"></option><option value="opus"></option><option value="haiku"></option></datalist>
            </label>
            <label class="stack">
              <span class="label">Modo padrão (vale para PRs abertas pela bandeja e para a pré-análise)</span>
              <span class="select">
                <select class="field" bind:value={config.defaultMode}>
                  <option value="code">Código: os arquivos críticos, com o código na tela</option>
                  <option value="architect">Arquiteto: a abordagem, os trade-offs e os diagramas</option>
                </select>
                {@render chevron()}
              </span>
            </label>

            <div class="prompts">
              <div class="prompt-block">
                <div class="prompt-head">
                  <span class="prompt-name">Instruções do modo Código</span>
                  {#if editingPrompt}
                    <button class="link quiet" onclick={() => (prompt = defaultPrompt)} disabled={prompt === defaultPrompt}>Restaurar padrão</button>
                  {/if}
                  <button class="link" aria-expanded={editingPrompt} onclick={() => (editingPrompt = !editingPrompt)}>
                    {editingPrompt ? "Fechar" : "Editar instruções"}
                  </button>
                </div>
                <p class="muted">
                  Diga o que o revisor deve priorizar, o tom e o idioma. As regras de formato da resposta (caminhos, linhas,
                  JSON) são do app e entram sempre, então nada aqui quebra o tour. Trocar as instruções gera uma análise nova.
                </p>
                {#if editingPrompt}
                  <textarea class="field prompt" rows="12" bind:value={prompt} aria-label="Instruções do modo Código" spellcheck="false"></textarea>
                {/if}
              </div>
              <div class="prompt-block">
                <div class="prompt-head">
                  <span class="prompt-name">Instruções do modo Arquiteto</span>
                  {#if editingArchitect}
                    <button class="link quiet" onclick={() => (architectPrompt = defaultArchitectPrompt)} disabled={architectPrompt === defaultArchitectPrompt}>
                      Restaurar padrão
                    </button>
                  {/if}
                  <button class="link" aria-expanded={editingArchitect} onclick={() => (editingArchitect = !editingArchitect)}>
                    {editingArchitect ? "Fechar" : "Editar instruções"}
                  </button>
                </div>
                <p class="muted">
                  O modo Arquiteto nunca cita função nem arquivo: essa regra e o formato dos diagramas são do app e entram sempre.
                  As instruções por repositório valem só para o modo Código.
                </p>
                {#if editingArchitect}
                  <textarea class="field prompt" rows="10" bind:value={architectPrompt} aria-label="Instruções do modo Arquiteto" spellcheck="false"></textarea>
                {/if}
              </div>
            </div>
          {:else}
            <h3>Sobre</h3>
            <p>Oddeye {version === "dev" ? "(desenvolvimento)" : `v${version}`}</p>
            <div class="update">
              <button class="btn" onclick={checkUpdate} disabled={checkingUpdate || version === "dev"}>
                {checkingUpdate ? "Procurando..." : "Procurar atualizações"}
              </button>
              {#if updateStatus}<span class="muted">{updateStatus}</span>{/if}
            </div>
            <p class="muted">Dados em <span class="mono path">{configDir}</span></p>
          {/if}
        </div>
      </div>

      <div class="modal-foot">
        {#if error}<span class="error-text foot-error">{error}</span>{/if}
        <button class="btn" onclick={onclose} disabled={saving}>Cancelar</button>
        <button class="btn primary" onclick={save} disabled={saving}>{saved ? "Salvo!" : saving ? "Salvando..." : "Salvar"}</button>
      </div>
    {/if}
  </div>
</div>

<style>
  /* fica acima do modal de review */
  .backdrop {
    z-index: calc(var(--z-modal) + 1);
  }
  .modal {
    max-width: 880px;
    height: 640px;
    text-align: left;
    /* sem isso o WebKitGTK desenha select e checkbox com o tema claro do sistema */
    color-scheme: dark;
  }
  header {
    display: flex;
    align-items: center;
    flex-shrink: 0;
    height: 56px;
    padding: 0 var(--sp-3) 0 var(--sp-5);
    border-bottom: 1px solid var(--border);
  }
  h2 {
    flex-grow: 1;
    margin: 0;
    font-size: var(--fs-lg);
    font-weight: 600;
  }
  .close {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border: none;
    border-radius: var(--r-sm);
    background: transparent;
    color: var(--text);
    transition: background var(--ease);
  }
  .close:hover {
    background: var(--selected);
  }
  svg {
    fill: none;
    stroke: currentColor;
    stroke-width: 1.5;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .loading {
    margin: 0;
    padding: var(--sp-5);
    color: var(--muted);
  }
  .layout {
    display: flex;
    flex-grow: 1;
    min-height: 0;
  }
  nav {
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
    gap: var(--sp-1);
    width: 180px;
    padding: var(--sp-3);
    border-right: 1px solid var(--border);
  }
  nav button {
    flex-shrink: 0;
    height: 36px;
    padding: 0 var(--sp-3);
    border: none;
    border-radius: var(--r-sm);
    background: transparent;
    color: var(--muted);
    font: inherit;
    text-align: left;
    white-space: nowrap;
    transition: background var(--ease), color var(--ease);
  }
  nav button:hover {
    color: var(--text);
  }
  nav button.on {
    background: var(--selected);
    color: var(--text);
    font-weight: 600;
  }
  .pane {
    display: flex;
    flex-direction: column;
    flex-grow: 1;
    gap: var(--sp-4);
    min-width: 0;
    padding: var(--sp-5);
    overflow: auto;
  }
  .pane > * {
    flex-shrink: 0;
  }
  @media (max-width: 640px) {
    .layout {
      flex-direction: column;
    }
    nav {
      flex-direction: row;
      width: auto;
      border-right: none;
      border-bottom: 1px solid var(--border);
      overflow-x: auto;
    }
  }
  h3 {
    margin: 0;
    font-size: var(--fs-md);
    font-weight: 600;
  }
  p {
    margin: 0;
  }
  .intro {
    display: flex;
    flex-direction: column;
    gap: var(--sp-1);
  }
  .muted {
    color: var(--muted);
  }
  small {
    display: block;
    color: var(--muted);
    font-size: inherit;
  }
  .warn {
    color: var(--yellow);
  }
  select.field {
    appearance: none;
    padding-right: var(--sp-6);
  }
  select option {
    background: var(--surface);
    color: var(--text);
  }
  .select {
    position: relative;
    display: block;
  }
  .select svg {
    position: absolute;
    top: 0;
    bottom: 0;
    right: var(--sp-3);
    margin: auto 0;
    color: var(--muted);
    pointer-events: none;
  }
  textarea.prompt {
    font-family: var(--mono);
    font-size: var(--fs-xs);
  }
  .picker {
    position: relative;
  }
  .suggestions {
    position: absolute;
    left: 0;
    right: 0;
    top: calc(100% + var(--sp-1));
    z-index: 2;
    list-style: none;
    margin: 0;
    padding: var(--sp-1);
    border: 1px solid var(--border-strong);
    border-radius: var(--r-sm);
    background: var(--surface);
    box-shadow: var(--shadow);
  }
  .suggestions button {
    width: 100%;
    padding: var(--sp-2) var(--sp-3);
    border: none;
    border-radius: var(--r-sm);
    background: transparent;
    color: var(--text);
    font-family: var(--mono);
    font-size: var(--fs-sm);
    text-align: left;
    transition: background var(--ease);
  }
  /* a primeira linha é a que o Enter adiciona; o destaque sai dela quando o mouse aponta outra */
  .suggestions button.first,
  .suggestions button:hover {
    background: var(--selected);
  }
  .suggestions:hover button.first:not(:hover) {
    background: transparent;
  }
  .tag {
    margin-left: var(--sp-2);
    font-family: var(--ui);
    font-size: var(--fs-xs);
    color: var(--muted);
  }
  .repos {
    list-style: none;
    margin: 0;
    padding: 0;
    border-bottom: 1px solid var(--border);
  }
  .repos li {
    display: flex;
    flex-direction: column;
    gap: var(--sp-4);
    padding: var(--sp-3) 0;
    border-top: 1px solid var(--border);
  }
  .repo-head {
    display: flex;
    align-items: baseline;
    gap: var(--sp-4);
  }
  .name {
    flex-grow: 1;
    min-width: 0;
    overflow-wrap: anywhere;
  }
  .repo-opts {
    display: flex;
    flex-direction: column;
    gap: var(--sp-3);
    padding-left: var(--sp-4);
    border-left: 1px solid var(--border-strong);
  }
  .checks {
    display: flex;
    flex-direction: column;
    gap: var(--sp-3);
  }
  .check {
    display: flex;
    align-items: flex-start;
    gap: var(--sp-3);
    cursor: pointer;
  }
  .check input {
    flex-shrink: 0;
    width: 16px;
    height: 16px;
    margin: 2px 0 0;
    cursor: pointer;
  }
  .stack {
    display: flex;
    flex-direction: column;
    gap: var(--sp-2);
  }
  .stack.narrow {
    max-width: 220px;
  }
  .prompts {
    border-bottom: 1px solid var(--border);
  }
  .prompt-block {
    display: flex;
    flex-direction: column;
    gap: var(--sp-2);
    padding: var(--sp-3) 0;
    border-top: 1px solid var(--border);
  }
  .prompt-head {
    display: flex;
    align-items: baseline;
    flex-wrap: wrap;
    gap: var(--sp-4);
  }
  .prompt-name {
    flex-grow: 1;
    font-weight: 600;
  }
  .update {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--sp-3);
  }
  .path {
    font-size: var(--fs-xs);
    overflow-wrap: anywhere;
  }
  .foot-error {
    flex: 1 1 240px;
    min-width: 0;
    max-height: 80px;
    overflow: auto;
  }
</style>
