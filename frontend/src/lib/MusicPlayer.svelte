<script lang="ts">
  import { api } from "./api";
  import { player } from "./player.svelte";

  let open = $state(false);
</script>

{#if player.unavailable}
  <div class="player off" class:open>
    <button class="title" onclick={() => (open = !open)} title="Por que o som está desligado">
      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 9v6h4l5 4V5L8 9zM17 9l5 6M22 9l-5 6" /></svg>
      <span class="name">Som indisponível</span>
    </button>
    {#if open}
      <div class="panel" role="dialog" aria-label="Som indisponível">
        <p class="hint">O sistema não tem o que o app precisa para tocar som. Falta:</p>
        <ul class="missing">
          {#each player.unavailable.missing as m (m)}<li>{m}</li>{/each}
        </ul>
        {#if player.unavailable.hint}
          <p class="hint">Instale com o comando abaixo e abra o Oddeye de novo:</p>
          <code class="cmd">{player.unavailable.hint}</code>
        {:else}
          <p class="hint">Instale o pacote de plugins "good" do GStreamer 1.0 da sua distribuição e abra o Oddeye de novo.</p>
        {/if}
        <p class="hint">Todo o resto do app funciona normalmente sem som.</p>
      </div>
    {/if}
  </div>
{:else}
<div class="player" class:open>
  <button
    class="title"
    onclick={() => {
      open = !open;
      if (open) void player.reload();
    }}
    title="Opções de som"
    aria-expanded={open}
  >
    <span class="name">{player.loading ? `${player.loading}: ${player.title}` : player.title}</span>
  </button>
  <button class="icon" onclick={() => player.prev()} title="Anterior" aria-label="Anterior">
    <svg viewBox="0 0 14 14" aria-hidden="true"><path d="M11 2.5l-6 4.5 6 4.5zM3 2.5v9" /></svg>
  </button>
  <button class="icon" onclick={() => player.toggle()} title={player.playing ? "Pausar" : "Tocar"} aria-label={player.playing ? "Pausar" : "Tocar"}>
    {#if player.playing}
      <svg viewBox="0 0 14 14" aria-hidden="true"><path d="M4.5 2.5v9M9.5 2.5v9" /></svg>
    {:else}
      <svg viewBox="0 0 14 14" aria-hidden="true"><path d="M4 2.5l7 4.5-7 4.5z" /></svg>
    {/if}
  </button>
  <button class="icon" onclick={() => player.next()} title="Próxima" aria-label="Próxima">
    <svg viewBox="0 0 14 14" aria-hidden="true"><path d="M3 2.5l6 4.5-6 4.5zM11 2.5v9" /></svg>
  </button>

  {#if open}
    <div class="panel" role="dialog" aria-label="Opções de som">
      <label class="row">
        <span>Volume</span>
        <input type="range" min="0" max="1" step="0.05" value={player.volume} oninput={(e) => player.setVolume(Number(e.currentTarget.value))} />
      </label>
      <label class="row check">
        <input type="checkbox" checked={player.sfx} onchange={() => player.toggleSfx()} />
        <span>Som suave ao clicar nos botões</span>
      </label>
      {#if player.error}<p class="err">{player.error}</p>{/if}
      <div class="lib">
        <p class="hint">
          {#if player.mode === "ambient"}
            Tocando a trilha ambiente. Busque músicas na biblioteca ou coloque arquivos (mp3, ogg, flac, wav, m4a) na pasta de músicas.
          {:else}
            {player.queueLabel}: {player.queue.length} faixa{player.queue.length === 1 ? "" : "s"}, tocando em ordem.
          {/if}
        </p>
        {#if player.loading}<p class="hint">{player.loading}...</p>{/if}
        <div class="actions">
          <button
            class="btn small"
            onclick={() => {
              open = false;
              player.libraryOpen = true;
            }}>Abrir biblioteca</button
          >
          <button class="link" onclick={() => api.openMusicFolder()}>Abrir pasta</button>
          <button class="link" onclick={() => player.reload()}>Recarregar</button>
        </div>
      </div>
    </div>
  {/if}
</div>
{/if}

<style>
  .player {
    position: relative;
    display: inline-flex;
    align-items: center;
    gap: 2px;
    height: 36px;
    padding: 0 4px 0 var(--sp-1);
    border: 1px solid var(--border);
    border-radius: var(--r-sm);
    background: var(--bg);
    max-width: 320px;
    transition: border-color var(--ease);
  }
  .player.open {
    border-color: var(--primary);
  }
  .icon {
    width: 28px;
    height: 28px;
    flex-shrink: 0;
    border: none;
    border-radius: 4px;
    background: none;
    color: var(--text);
    display: grid;
    place-items: center;
    padding: 0;
    transition: background var(--ease);
  }
  .icon:hover {
    background: var(--selected);
  }
  svg {
    width: 14px;
    height: 14px;
    flex-shrink: 0;
    fill: none;
    stroke: currentColor;
    stroke-width: 1.5;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .title {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
    min-width: 0;
    height: 28px;
    padding: 0 var(--sp-2);
    border: none;
    background: none;
    color: var(--muted);
    font-size: var(--fs-sm);
    transition: color var(--ease);
  }
  .title:hover,
  .open .title {
    color: var(--text);
  }
  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 180px;
  }
  .panel {
    position: absolute;
    bottom: calc(100% + 8px);
    right: 0;
    z-index: var(--z-popover);
    width: 340px;
    display: flex;
    flex-direction: column;
    gap: var(--sp-4);
    padding: var(--sp-4);
    border: 1px solid var(--border-strong);
    border-radius: var(--r-md);
    background: var(--surface);
    box-shadow: var(--shadow);
    font-size: var(--fs-sm);
  }
  :global(.home-player) .panel {
    bottom: auto;
    top: calc(100% + 8px);
  }
  .row {
    display: flex;
    align-items: center;
    gap: var(--sp-3);
  }
  .row input[type="range"] {
    flex: 1;
  }
  .err {
    margin: 0;
    color: var(--red);
  }
  .lib {
    display: flex;
    flex-direction: column;
    gap: var(--sp-2);
  }
  .hint {
    margin: 0;
    color: var(--muted);
  }
  .missing {
    margin: 0;
    padding-left: 18px;
  }
  code {
    display: block;
    font: var(--fs-xs) var(--mono);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    direction: rtl;
    text-align: left;
  }
  code.cmd {
    user-select: all;
    white-space: normal;
    word-break: break-word;
    direction: ltr;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--sp-4);
  }
</style>
