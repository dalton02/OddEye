<script lang="ts">
  import { onMount } from "svelte";
  import { api, errorMessage, type Deps, type DeviceCode } from "./api";

  interface Props {
    deps: Deps;
    onrecheck: () => Promise<void>;
  }

  let { deps, onrecheck }: Props = $props();

  let authorizing = $state(false);
  let code = $state<DeviceCode | null>(null);
  let copied = $state(false);
  let error = $state("");

  onMount(() => api.onDeviceCode((d) => (code = d)));

  const needsAuth = $derived(deps.ghInstalled && (!!deps.ghError || deps.missingScopes.length > 0));

  async function authorize(): Promise<void> {
    authorizing = true;
    error = "";
    code = null;
    try {
      await api.authorizeGitHub();
      await onrecheck();
    } catch (e) {
      error = errorMessage(e);
    } finally {
      authorizing = false;
      code = null;
    }
  }

  async function copy(): Promise<void> {
    if (!code) return;
    try {
      await navigator.clipboard.writeText(code.code);
      copied = true;
      setTimeout(() => (copied = false), 1500);
    } catch {
      /* sem acesso à área de transferência: o código continua visível para digitar */
    }
  }
</script>

{#snippet mark(done: boolean, n: number)}
  {#if done}
    <svg class="check" role="img" aria-label="Concluído" width="16" height="16" viewBox="0 0 16 16">
      <path d="M3 8.5l3.2 3L13 4.5" />
    </svg>
  {:else}
    <span class="num" aria-hidden="true">{n}</span>
  {/if}
{/snippet}

<div class="setup">
  <h2>Falta pouco para começar</h2>
  <ol>
    <li>
      {@render mark(deps.ghInstalled, 1)}
      {#if deps.ghInstalled}
        <div class="item done">
          <strong>GitHub CLI</strong>
          <span class="desc">instalado</span>
        </div>
      {:else}
        <div class="item">
          <div class="text">
            <strong>GitHub CLI</strong>
            <span class="desc">O app usa o <code>gh</code> para ler PRs e publicar reviews na sua conta.</span>
            <button class="link" onclick={() => api.openURL("https://cli.github.com")}>Instalar o gh</button>
          </div>
        </div>
      {/if}
    </li>
    <li>
      {@render mark(deps.ghInstalled && !needsAuth, 2)}
      {#if !deps.ghInstalled}
        <div class="item">
          <div class="text">
            <strong>Acesso ao GitHub</strong>
            <span class="desc">depois de instalar o gh</span>
          </div>
        </div>
      {:else if !needsAuth}
        <div class="item done">
          <strong>Acesso ao GitHub</strong>
          <span class="desc">autorizado</span>
        </div>
      {:else}
        <div class="item">
          <div class="text">
            <strong>Acesso ao GitHub</strong>
            <span class="desc">
              {deps.ghError ? "Conecte sua conta" : `Faltam as permissões ${deps.missingScopes.join(", ")}`}:
              ler PRs (inclusive privadas), comentar e aprovar. Tudo sai na sua conta, só quando você clicar em enviar.
            </span>
          </div>
          {#if code}
            <div class="line">
              <span>Digite este código no GitHub:</span>
              <button class="code-value" onclick={copy} title="Copiar">{code.code}</button>
              <span class="desc">{copied ? "copiado!" : "clique para copiar"}</span>
            </div>
          {/if}
          <div class="line">
            <button class="btn primary" onclick={authorize} disabled={authorizing}>
              {authorizing ? "Aguardando o GitHub..." : deps.ghError ? "Conectar ao GitHub" : "Autorizar permissões"}
            </button>
            {#if code}
              <button class="link" onclick={() => code && api.openURL(code.url)}>Abrir a página do GitHub</button>
            {/if}
            {#if authorizing}<button class="link quiet" onclick={() => api.cancelAuthorize()}>Cancelar</button>{/if}
          </div>
        </div>
      {/if}
    </li>
    <li>
      {@render mark(!deps.claudeError, 3)}
      {#if !deps.claudeError}
        <div class="item done">
          <strong>Claude Code</strong>
          <span class="desc">instalado (usa o login que você já tem, sem chave de API)</span>
        </div>
      {:else}
        <div class="item">
          <div class="text">
            <strong>Claude Code</strong>
            <span class="desc">
              O revisor roda no seu Claude Code. Instale, abra <code>claude</code> no terminal uma vez e faça login.
            </span>
            <button class="link" onclick={() => api.openURL("https://claude.com/claude-code")}>Instalar o Claude Code</button>
          </div>
        </div>
      {/if}
    </li>
  </ol>
  {#if error}<p class="error-text">{error}</p>{/if}
  <div class="foot">
    <button class="btn" onclick={onrecheck}>Verificar de novo</button>
  </div>
</div>

<style>
  .setup {
    display: flex;
    flex-direction: column;
    width: 100%;
    text-align: left;
  }
  h2 {
    margin: 0;
    padding-bottom: var(--sp-3);
    border-bottom: 1px solid var(--border);
    font-size: var(--fs-md);
    font-weight: 600;
  }
  ol {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  li {
    display: flex;
    gap: var(--sp-4);
    padding: var(--sp-4) 0;
    border-bottom: 1px solid var(--border);
  }
  .check {
    flex-shrink: 0;
    margin-top: 2px;
    fill: none;
    stroke: var(--green);
    stroke-width: 1.5;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .num {
    flex-shrink: 0;
    width: 16px;
    color: var(--muted);
    text-align: center;
  }
  .item {
    display: flex;
    flex-direction: column;
    flex-grow: 1;
    gap: var(--sp-3);
    min-width: 0;
  }
  .item.done {
    flex-direction: row;
    flex-wrap: wrap;
    align-items: baseline;
    column-gap: var(--sp-3);
    row-gap: 0;
  }
  .text {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--sp-1);
  }
  strong {
    font-weight: 600;
  }
  .desc {
    color: var(--muted);
  }
  code {
    font-family: var(--mono);
    font-size: var(--fs-xs);
    color: var(--text);
  }
  .line {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--sp-4);
  }
  .code-value {
    height: 44px;
    padding: 0 var(--sp-4);
    border: 1px solid var(--border-strong);
    border-radius: var(--r-sm);
    background: var(--surface);
    color: var(--text);
    font-family: var(--mono);
    font-size: var(--fs-lg);
    letter-spacing: 0.08em;
    transition: background var(--ease);
  }
  .code-value:hover {
    background: var(--selected);
  }
  .error-text {
    margin: var(--sp-4) 0 0;
  }
  .foot {
    padding-top: var(--sp-4);
  }
</style>
