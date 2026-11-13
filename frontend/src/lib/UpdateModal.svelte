<script lang="ts">
  import { api, type UpdateInfo, type UpdateProgress } from "./api";

  interface Props {
    info: UpdateInfo;
    onclose: () => void;
  }

  let { info, onclose }: Props = $props();

  let progress = $state<UpdateProgress | null>(null);
  const busy = $derived(!!progress && progress.stage !== "error");

  const percent = $derived(
    progress?.stage === "downloading" && progress.total > 0 ? Math.min(100, Math.round((progress.done / progress.total) * 100)) : null
  );

  const status = $derived.by((): string => {
    switch (progress?.stage) {
      case "downloading":
        return percent === null ? "Baixando…" : `Baixando… ${percent}%`;
      case "installing":
        return info.needsPassword ? "Instalando… confirme a senha na janela do sistema." : "Instalando…";
      case "restarting":
        return "Reiniciando…";
      default:
        return "";
    }
  });

  $effect(() => api.onUpdateProgress((p) => (progress = p)));

  async function install(): Promise<void> {
    progress = { stage: "downloading", done: 0, total: 0 };
    try {
      await api.installUpdate();
    } catch (e) {
      /* o backend já emitiu o evento de erro com os detalhes; isto cobre falha da própria chamada */
      if (progress?.stage !== "error") progress = { stage: "error", done: 0, total: 0, error: String(e) };
    }
  }

  async function skip(): Promise<void> {
    await api.skipUpdate(info.version);
    onclose();
  }

  function onKey(e: KeyboardEvent): void {
    if (e.key === "Escape" && !busy) onclose();
  }
</script>

<svelte:window onkeydown={onKey} />

<div class="backdrop" role="presentation" onclick={(e) => e.target === e.currentTarget && !busy && onclose()}>
  <div class="modal" role="dialog" aria-modal="true" aria-labelledby="update-title">
    <h2 id="update-title">Nova versão disponível: v{info.version}</h2>
    <p class="sub">Você está na v{info.current}.</p>

    {#if info.notes}
      <section>
        <h3>O que mudou</h3>
        <p class="notes">{info.notes}</p>
      </section>
    {/if}

    {#if progress && progress.stage !== "error"}
      <div class="progress" role="status">
        <div class="bar"><div class="fill" class:indeterminate={percent === null} style:width={percent === null ? "40%" : `${percent}%`}></div></div>
        <p>{status}</p>
      </div>
    {:else if progress?.stage === "error"}
      <p class="error">
        {progress.manual
          ? "Não consegui atualizar sozinho neste computador. Baixe a versão nova pela página do release."
          : `A atualização falhou: ${progress.error}`}
      </p>
    {:else if !info.automatic}
      <p class="muted">Este jeito de instalação não atualiza sozinho. Baixe a versão nova pela página do release.</p>
    {:else if info.needsPassword}
      <p class="muted">O sistema vai pedir sua senha uma vez para instalar o pacote. Depois o Oddeye reinicia sozinho.</p>
    {:else}
      <p class="muted">O Oddeye baixa a versão nova, instala e reinicia sozinho. Seus rascunhos ficam salvos.</p>
    {/if}

    <footer>
      {#if !busy}
        <button class="link" onclick={skip}>Pular esta versão</button>
        <span class="spacer"></span>
        <button class="btn" onclick={onclose}>Depois</button>
        {#if info.automatic && !progress?.manual}
          <button class="btn primary" onclick={install}>{progress?.stage === "error" ? "Tentar de novo" : "Atualizar agora"}</button>
        {/if}
        {#if !info.automatic || progress?.stage === "error"}
          <button class="btn" class:primary={!info.automatic || progress?.manual} onclick={() => api.openReleasePage()}>Baixar manualmente</button>
        {/if}
      {/if}
    </footer>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 60;
    background: rgba(2, 3, 10, 0.82);
    display: grid;
    place-items: center;
    padding: 24px;
  }
  .modal {
    width: min(560px, 100%);
    max-height: 90vh;
    overflow: auto;
    background: var(--panel-strong);
    border: 1px solid var(--border);
    border-radius: 16px;
    padding: 24px;
    box-shadow: 0 30px 90px rgba(0, 0, 0, 0.6);
  }
  h2 {
    margin: 0;
    font-size: 20px;
  }
  .sub {
    margin: 4px 0 18px;
    color: var(--muted);
    font-size: 13px;
  }
  h3 {
    margin: 0 0 8px;
    font-size: 13px;
    color: var(--muted);
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }
  section {
    margin-bottom: 18px;
  }
  .notes {
    margin: 0;
    max-height: 220px;
    overflow: auto;
    white-space: pre-wrap;
    font-size: 13px;
    line-height: 1.5;
    padding: 12px;
    border: 1px solid var(--border);
    border-radius: 10px;
    background: rgba(255, 255, 255, 0.02);
  }
  .muted {
    color: var(--muted);
    font-size: 13px;
    margin: 0 0 18px;
  }
  .error {
    color: var(--red);
    font-size: 13px;
    margin: 0 0 18px;
  }
  .progress {
    margin-bottom: 18px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    font-size: 13px;
  }
  .progress p {
    margin: 0;
    color: var(--muted);
  }
  .bar {
    height: 6px;
    border-radius: 3px;
    background: rgba(255, 255, 255, 0.08);
    overflow: hidden;
  }
  .fill {
    height: 100%;
    background: var(--accent);
    transition: width 0.2s ease;
  }
  .fill.indeterminate {
    animation: slide 1.2s ease-in-out infinite;
  }
  @keyframes slide {
    0% {
      transform: translateX(-100%);
    }
    100% {
      transform: translateX(250%);
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .fill.indeterminate {
      animation: none;
    }
  }
  footer {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
  }
  .spacer {
    flex: 1;
  }
  .link {
    background: none;
    border: none;
    padding: 0;
    color: var(--muted);
    font-size: 13px;
    text-decoration: underline;
  }
  .link:hover {
    color: var(--text);
  }
</style>
