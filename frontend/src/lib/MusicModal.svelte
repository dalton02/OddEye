<script lang="ts">
  import { onMount } from "svelte";
  import { api, errorMessage, type ArchiveItem, type Playlist, type TorrentFile, type TorrentInfo } from "./api";
  import { filesOfFormat, fmtBytes, folderTrack, formatsOf, percent, torrentTrack, type Track } from "./music";
  import { player } from "./player.svelte";

  interface Props {
    onclose: () => void;
  }

  let { onclose }: Props = $props();

  type Tab = "search" | "downloads" | "playlists";
  const tabs: { id: Tab; label: string }[] = [
    { id: "search", label: "Buscar" },
    { id: "downloads", label: "Downloads" },
    { id: "playlists", label: "Playlists" }
  ];
  let tab = $state<Tab>("search");

  /* ---------- busca ---------- */
  let query = $state("");
  let results = $state<ArchiveItem[]>([]);
  let total = $state(0);
  let page = $state(1);
  let searching = $state(false);
  let searched = $state(false);
  let searchError = $state("");

  interface Album {
    key: string;
    title: string;
    creator: string;
    hash: string;
    files: TorrentFile[];
    format: string;
    loading: boolean;
    error: string;
  }
  let album = $state<Album | null>(null);
  let magnet = $state("");
  let adding = $state(false);
  let addError = $state("");
  /** playlist que recebe as faixas do "+"; "__new" cria uma com o nome do álbum */
  let target = $state("__new");
  let notice = $state("");

  const albumTracks = $derived(
    album && album.hash ? filesOfFormat(album.files, album.format).map((f) => torrentTrack(album!.hash, f.path, album!.creator)) : []
  );
  const albumFiles = $derived(album ? new Map(album.files.map((f) => [f.path, f])) : new Map<string, TorrentFile>());

  async function search(p = 1): Promise<void> {
    if (!query.trim()) return;
    searching = true;
    searchError = "";
    try {
      const res = await api.searchArchive(query, p);
      results = p === 1 ? res.items : [...results, ...res.items];
      total = res.total;
      page = p;
      searched = true;
    } catch (e) {
      searchError = errorMessage(e);
    } finally {
      searching = false;
    }
  }

  async function openAlbum(key: string, title: string, creator: string, add: () => Promise<TorrentInfo>): Promise<void> {
    if (album?.key === key) {
      album = null;
      return;
    }
    notice = "";
    album = { key, title, creator, hash: "", files: [], format: "", loading: true, error: "" };
    try {
      const info = await add();
      const files = await api.torrentFiles(info.infoHash);
      if (album?.key !== key) return;
      album.hash = info.infoHash;
      album.files = files;
      album.format = formatsOf(files)[0] ?? "";
      if (title === "") album.title = info.title;
    } catch (e) {
      if (album?.key === key) album.error = errorMessage(e);
    } finally {
      if (album?.key === key) album.loading = false;
    }
  }

  function openItem(item: ArchiveItem): void {
    void openAlbum(item.id, item.title, item.creator, () => api.addArchiveItem(item.id, item.title));
  }

  async function addManual(fromFile: boolean): Promise<void> {
    adding = true;
    addError = "";
    try {
      const info = fromFile ? await api.chooseTorrentFile() : await api.addTorrent(magnet);
      if (!info) return;
      magnet = "";
      results = [];
      searched = false;
      album = null;
      await openAlbum("t:" + info.infoHash, info.title, "", async () => info);
    } catch (e) {
      addError = errorMessage(e);
    } finally {
      adding = false;
    }
  }

  async function refreshAlbum(): Promise<void> {
    if (!album?.hash) return;
    const key = album.key;
    try {
      const files = await api.torrentFiles(album.hash);
      if (album?.key === key) album.files = files;
    } catch {
      /* torrent removido no meio: a próxima ação mostra o erro */
    }
  }

  function playAlbum(i: number): void {
    if (!album) return;
    player.playQueue(albumTracks, i, album.title, null);
  }

  /* Adiciona e já começa a baixar em segundo plano, para a playlist tocar sem espera depois. */
  async function addToPlaylist(tracks: Track[]): Promise<void> {
    if (!tracks.length) return;
    notice = "";
    try {
      let id = target;
      if (id === "__new" || !player.playlists.some((p) => p.id === id)) {
        const created = await api.createPlaylist(album?.title || "Nova playlist");
        player.playlistChanged(created);
        id = created.id;
      }
      const p = await api.addToPlaylist(id, tracks);
      player.playlistChanged(p);
      target = p.id;
      notice = `${tracks.length === 1 ? "Faixa adicionada" : `${tracks.length} faixas adicionadas`} em "${p.name}".`;
      for (const t of tracks) {
        if (t.source === "torrent") void api.downloadTrack(t.infoHash ?? "", t.path ?? "", false).catch(() => {});
      }
      void refreshAlbum();
    } catch (e) {
      notice = errorMessage(e);
    }
  }

  /* ---------- downloads ---------- */
  let torrents = $state<TorrentInfo[]>([]);
  let confirmRemove = $state<string | null>(null);
  let downloadsError = $state("");

  async function loadTorrents(): Promise<void> {
    try {
      torrents = await api.listTorrents();
    } catch (e) {
      downloadsError = errorMessage(e);
    }
  }

  async function removeTorrent(hash: string, deleteFiles: boolean): Promise<void> {
    confirmRemove = null;
    try {
      await api.removeTorrent(hash, deleteFiles);
      if (album?.hash === hash) album = null;
      await loadTorrents();
    } catch (e) {
      downloadsError = errorMessage(e);
    }
  }

  /* ---------- playlists ---------- */
  /** "" é a pasta local, que funciona como uma playlist só de leitura */
  let selected = $state("");
  let newName = $state("");
  let nameDraft = $state("");
  let confirmDelete = $state(false);
  let listError = $state("");

  const selectedList = $derived<Playlist | null>(player.playlists.find((p) => p.id === selected) ?? null);
  const selectedTracks = $derived<Track[]>(selected === "" ? player.library.tracks.map(folderTrack) : (selectedList?.tracks ?? []));

  function select(id: string): void {
    selected = id;
    confirmDelete = false;
    listError = "";
    nameDraft = player.playlists.find((p) => p.id === id)?.name ?? "";
  }

  async function guard(fn: () => Promise<void>): Promise<void> {
    listError = "";
    try {
      await fn();
    } catch (e) {
      listError = errorMessage(e);
    }
  }

  const createList = (): Promise<void> =>
    guard(async () => {
      const p = await api.createPlaylist(newName);
      player.playlistChanged(p);
      newName = "";
      select(p.id);
    });

  const renameList = (): Promise<void> =>
    guard(async () => {
      if (!selectedList || nameDraft.trim() === selectedList.name) return;
      await api.renamePlaylist(selectedList.id, nameDraft);
      player.playlistChanged({ ...selectedList, name: nameDraft.trim() });
    });

  const deleteList = (): Promise<void> =>
    guard(async () => {
      if (!selectedList) return;
      await api.deletePlaylist(selectedList.id);
      player.playlistDeleted(selectedList.id);
      select("");
    });

  const moveTrack = (from: number, to: number): Promise<void> =>
    guard(async () => {
      if (!selectedList || to < 0 || to >= selectedList.tracks.length) return;
      player.playlistChanged(await api.movePlaylistTrack(selectedList.id, from, to));
    });

  const removeTrack = (trackID: string): Promise<void> =>
    guard(async () => {
      if (!selectedList) return;
      player.playlistChanged(await api.removeFromPlaylist(selectedList.id, trackID));
    });

  function isPlaying(listId: string | null, i: number): boolean {
    return player.playing && player.playlistId === listId && player.index === i;
  }

  onMount(() => {
    void loadTorrents();
    void player.reload();
    if (player.playlists.length) target = player.playlistId || player.playlists[0].id;
    const off = api.onTorrentProgress((list) => {
      torrents = list;
      if (album?.hash && list.some((t) => t.infoHash === album!.hash)) void refreshAlbum();
    });
    return off;
  });

  function onKey(e: KeyboardEvent): void {
    if (e.key === "Escape") onclose();
  }
</script>

<svelte:window onkeydown={onKey} />

{#snippet icon(d: string)}
  <svg aria-hidden="true" width="14" height="14" viewBox="0 0 14 14"><path {d} /></svg>
{/snippet}

{#snippet trackStatus(f: TorrentFile | undefined)}
  {#if f?.complete}
    <span class="tag ok" title="Baixada">baixada</span>
  {:else if f?.wanted}
    <span class="tag">{percent(f.done, f.size)}%</span>
  {/if}
{/snippet}

<div class="backdrop" role="presentation" onclick={(e) => e.target === e.currentTarget && onclose()}>
  <div class="modal" role="dialog" aria-modal="true" aria-labelledby="music-title">
    <header>
      <h2 id="music-title">Música</h2>
      {#if player.mode === "queue"}
        <span class="now" title={player.queueLabel}>
          {player.loading || (player.playing ? "Tocando" : "Pausado")}: {player.title}
        </span>
      {/if}
      <button class="close" onclick={onclose} aria-label="Fechar">{@render icon("M3 3l8 8M11 3l-8 8")}</button>
    </header>

    <div class="layout">
      <nav aria-label="Seções">
        {#each tabs as t (t.id)}
          <button class:on={tab === t.id} aria-current={tab === t.id ? "page" : undefined} onclick={() => (tab = t.id)}>
            {t.label}
          </button>
        {/each}
      </nav>

      <div class="pane">
        {#if tab === "search"}
          <form
            class="bar"
            onsubmit={(e) => {
              e.preventDefault();
              void search(1);
            }}
          >
            <input class="field grow" placeholder="Artista, álbum, gênero..." bind:value={query} aria-label="Buscar no Internet Archive" />
            <button class="btn primary" disabled={searching || !query.trim()}>{searching ? "Buscando..." : "Buscar"}</button>
          </form>
          <p class="muted small">
            Busca no Internet Archive: shows ao vivo, netlabels, Creative Commons e domínio público. As faixas chegam por torrent, com
            o servidor do archive.org como fonte reserva.
          </p>

          <details class="manual">
            <summary>Colar magnet ou abrir .torrent</summary>
            <div class="bar">
              <input class="field grow mono" placeholder="magnet:?xt=urn:btih:..." bind:value={magnet} aria-label="Link magnet" />
              <button class="btn" disabled={adding || !magnet.trim()} onclick={() => addManual(false)}>Adicionar</button>
              <button class="btn" disabled={adding} onclick={() => addManual(true)}>Abrir .torrent</button>
            </div>
            {#if adding}<p class="muted small">Buscando os metadados do torrent...</p>{/if}
            {#if addError}<p class="err">{addError}</p>{/if}
          </details>

          {#if searchError}<p class="err">{searchError}</p>{/if}

          {#if album && album.key.startsWith("t:")}
            {@render albumBody(album)}
          {/if}

          {#if searched && !results.length}
            <p class="muted">Nada encontrado para "{query}".</p>
          {/if}

          <ul class="results">
            {#each results as item (item.id)}
              <li class:open={album?.key === item.id}>
                <button class="result" onclick={() => openItem(item)} aria-expanded={album?.key === item.id}>
                  <span class="r-title">{item.title}</span>
                  <span class="r-meta">
                    {[item.creator, item.date, item.license ? "CC/licença aberta" : ""].filter(Boolean).join(" · ")}
                  </span>
                </button>
                {#if album?.key === item.id}
                  {@render albumBody(album)}
                {/if}
              </li>
            {/each}
          </ul>
          {#if results.length && results.length < total}
            <button class="btn more" disabled={searching} onclick={() => search(page + 1)}>
              {searching ? "Carregando..." : `Mais resultados (${results.length} de ${total})`}
            </button>
          {/if}
        {:else if tab === "downloads"}
          <div class="intro">
            <h3>Downloads</h3>
            <p class="muted">Só as faixas que você toca ou põe numa playlist são baixadas. O app não compartilha (seed) depois de terminar.</p>
          </div>
          {#if downloadsError}<p class="err">{downloadsError}</p>{/if}
          {#if !torrents.length}
            <p class="muted">Nenhum torrent ainda. Toque ou adicione uma faixa na aba Buscar.</p>
          {/if}
          <ul class="downloads">
            {#each torrents as t (t.infoHash)}
              <li>
                <div class="d-head">
                  <span class="r-title" title={t.name}>{t.title}</span>
                  {#if confirmRemove === t.infoHash}
                    <button class="link danger" onclick={() => removeTorrent(t.infoHash, true)}>Remover e apagar arquivos</button>
                    <button class="link" onclick={() => removeTorrent(t.infoHash, false)}>Só remover da lista</button>
                    <button class="link" onclick={() => (confirmRemove = null)}>Cancelar</button>
                  {:else}
                    <button class="link" onclick={() => (confirmRemove = t.infoHash)}>Remover</button>
                  {/if}
                </div>
                <div class="progress" role="progressbar" aria-valuenow={percent(t.done, t.size)} aria-valuemin={0} aria-valuemax={100}>
                  <span style:width="{percent(t.done, t.size)}%"></span>
                </div>
                <span class="r-meta">
                  {t.complete}/{t.wanted} faixas · {fmtBytes(t.done)} de {fmtBytes(t.size)}
                  {#if t.done < t.size}· {fmtBytes(t.rate)}/s · {t.peers} peer{t.peers === 1 ? "" : "s"}{/if}
                </span>
              </li>
            {/each}
          </ul>
        {:else}
          <div class="lists">
            <aside>
              <button class="list-item" class:on={selected === ""} onclick={() => select("")}>
                <span>Pasta local</span>
                <span class="count">{player.library.tracks.length}</span>
              </button>
              {#each player.playlists as p (p.id)}
                <button class="list-item" class:on={selected === p.id} onclick={() => select(p.id)}>
                  <span>{p.name}</span>
                  <span class="count">{p.tracks.length}</span>
                </button>
              {/each}
              <form
                class="new"
                onsubmit={(e) => {
                  e.preventDefault();
                  void createList();
                }}
              >
                <input class="field" placeholder="Nova playlist" bind:value={newName} aria-label="Nome da nova playlist" />
                <button class="btn square" disabled={!newName.trim()} aria-label="Criar playlist" title="Criar playlist">
                  {@render icon("M7 3v8M3 7h8")}
                </button>
              </form>
            </aside>

            <section>
              <div class="list-head">
                {#if selectedList}
                  <input
                    class="field grow name"
                    bind:value={nameDraft}
                    onblur={renameList}
                    onkeydown={(e) => e.key === "Enter" && e.currentTarget.blur()}
                    aria-label="Nome da playlist"
                  />
                {:else}
                  <h3 class="grow">Pasta local</h3>
                {/if}
                <button class="btn primary" disabled={!selectedTracks.length} onclick={() => player.playPlaylist(selected)}>Tocar</button>
                {#if selectedList}
                  {#if confirmDelete}
                    <button class="btn danger" onclick={deleteList}>Confirmar exclusão</button>
                    <button class="btn" onclick={() => (confirmDelete = false)}>Cancelar</button>
                  {:else}
                    <button class="btn" onclick={() => (confirmDelete = true)}>Excluir</button>
                  {/if}
                {/if}
              </div>
              {#if listError}<p class="err">{listError}</p>{/if}
              {#if selected === ""}
                <p class="muted small" title={player.library.dir}>
                  Arquivos de {player.library.dir || "a pasta de músicas"}.
                  <button class="link" onclick={() => api.openMusicFolder()}>Abrir pasta</button>
                  <button class="link" onclick={() => player.chooseFolder()}>Escolher outra</button>
                </p>
              {/if}
              {#if !selectedTracks.length}
                <p class="muted">
                  {selected === "" ? "A pasta está vazia." : "Playlist vazia. Adicione faixas pela aba Buscar."}
                </p>
              {/if}
              <ol class="tracks">
                {#each selectedTracks as t, i (t.id)}
                  <li class:playing={isPlaying(selected, i)}>
                    <button class="icon" onclick={() => player.playPlaylist(selected, i)} aria-label="Tocar {t.title}" title="Tocar">
                      {@render icon("M4 2.5l7 4.5-7 4.5z")}
                    </button>
                    <span class="t-title" title={t.path ?? t.name}>{t.title}</span>
                    <span class="tag">{t.source === "torrent" ? "torrent" : "pasta"}</span>
                    {#if selectedList}
                      <button class="icon" disabled={i === 0} onclick={() => moveTrack(i, i - 1)} aria-label="Subir" title="Subir">
                        {@render icon("M7 11V3M3.5 6.5L7 3l3.5 3.5")}
                      </button>
                      <button
                        class="icon"
                        disabled={i === selectedTracks.length - 1}
                        onclick={() => moveTrack(i, i + 1)}
                        aria-label="Descer"
                        title="Descer"
                      >
                        {@render icon("M7 3v8M3.5 7.5L7 11l3.5-3.5")}
                      </button>
                      <button class="icon" onclick={() => removeTrack(t.id)} aria-label="Remover da playlist" title="Remover da playlist">
                        {@render icon("M3.5 3.5l7 7M10.5 3.5l-7 7")}
                      </button>
                    {/if}
                  </li>
                {/each}
              </ol>
            </section>
          </div>
        {/if}
      </div>
    </div>
  </div>
</div>

{#snippet albumBody(a: Album)}
  <div class="album">
    {#if a.loading}
      <p class="muted small">Buscando a lista de faixas do torrent...</p>
    {:else if a.error}
      <p class="err">{a.error}</p>
    {:else}
      <div class="album-bar">
        {#if a.key.startsWith("t:")}<strong class="grow">{a.title}</strong>{/if}
        {#if formatsOf(a.files).length > 1}
          <label class="inline">
            Formato
            <select class="select" bind:value={a.format}>
              {#each formatsOf(a.files) as f (f)}<option value={f}>{f.toUpperCase()}</option>{/each}
            </select>
          </label>
        {/if}
        <label class="inline">
          Adicionar em
          <select class="select" bind:value={target}>
            {#each player.playlists as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
            <option value="__new">Nova playlist "{a.title.slice(0, 30)}"</option>
          </select>
        </label>
        <button class="btn" onclick={() => addToPlaylist(albumTracks)}>Adicionar todas</button>
        <button class="btn primary" onclick={() => playAlbum(0)}>Tocar álbum</button>
      </div>
      {#if notice}<p class="muted small">{notice}</p>{/if}
      <ol class="tracks">
        {#each albumTracks as t, i (t.id)}
          {@const f = albumFiles.get(t.path ?? "")}
          <li class:playing={isPlaying(null, i) && player.queueLabel === a.title}>
            <button class="icon" onclick={() => playAlbum(i)} aria-label="Tocar {t.title}" title="Tocar">{@render icon("M4 2.5l7 4.5-7 4.5z")}</button>
            <span class="t-title" title={t.path}>{t.title}</span>
            {@render trackStatus(f)}
            <span class="size">{fmtBytes(f?.size ?? 0)}</span>
            <button class="icon" onclick={() => addToPlaylist([t])} aria-label="Adicionar à playlist" title="Adicionar à playlist">
              {@render icon("M7 3v8M3 7h8")}
            </button>
          </li>
        {/each}
      </ol>
    {/if}
  </div>
{/snippet}

<style>
  .modal {
    max-width: 960px;
    height: 680px;
    text-align: left;
    /* sem isso o WebKitGTK desenha select e checkbox com o tema claro do sistema */
    color-scheme: dark;
  }
  header {
    display: flex;
    align-items: center;
    gap: var(--sp-4);
    flex-shrink: 0;
    height: 56px;
    padding: 0 var(--sp-3) 0 var(--sp-5);
    border-bottom: 1px solid var(--border);
  }
  h2 {
    margin: 0;
    font-size: var(--fs-lg);
    font-weight: 600;
  }
  .now {
    flex-grow: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--muted);
    font-size: var(--fs-sm);
  }
  .close {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    margin-left: auto;
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
    width: 160px;
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
  .small {
    font-size: var(--fs-sm);
  }
  .mono {
    font-family: var(--mono);
    font-size: var(--fs-xs);
  }
  .err {
    color: var(--red);
    font-size: var(--fs-sm);
  }
  .bar {
    display: flex;
    gap: var(--sp-2);
  }
  .grow {
    flex-grow: 1;
    min-width: 0;
  }
  .manual summary {
    color: var(--muted);
    font-size: var(--fs-sm);
    cursor: pointer;
    transition: color var(--ease);
  }
  .manual summary:hover {
    color: var(--text);
  }
  .manual[open] {
    display: flex;
    flex-direction: column;
    gap: var(--sp-2);
  }
  ul,
  ol {
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .results {
    display: flex;
    flex-direction: column;
    border-top: 1px solid var(--border);
  }
  .results > li {
    border-bottom: 1px solid var(--border);
  }
  .result {
    display: flex;
    flex-direction: column;
    gap: 2px;
    width: 100%;
    padding: var(--sp-3) var(--sp-2);
    border: none;
    background: none;
    color: var(--text);
    font: inherit;
    text-align: left;
    transition: background var(--ease);
  }
  .result:hover,
  .open .result {
    background: var(--surface-2);
  }
  .r-title {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-weight: 500;
  }
  .r-meta {
    color: var(--muted);
    font-size: var(--fs-xs);
  }
  .more {
    align-self: center;
  }
  .album {
    display: flex;
    flex-direction: column;
    gap: var(--sp-3);
    padding: var(--sp-3) var(--sp-2) var(--sp-4);
  }
  .album-bar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--sp-2);
  }
  .inline {
    display: inline-flex;
    align-items: center;
    gap: var(--sp-2);
    color: var(--muted);
    font-size: var(--fs-sm);
  }
  .inline .select {
    max-width: 220px;
  }
  .tracks {
    display: flex;
    flex-direction: column;
  }
  .tracks li {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
    min-height: 32px;
    padding: 0 var(--sp-1);
    border-radius: var(--r-sm);
    font-size: var(--fs-sm);
    transition: background var(--ease);
  }
  .tracks li:hover {
    background: var(--surface-2);
  }
  .tracks li.playing .t-title {
    color: var(--primary);
    font-weight: 600;
  }
  .t-title {
    flex-grow: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--text-2);
  }
  .size {
    width: 56px;
    color: var(--muted);
    font-size: var(--fs-xs);
    text-align: right;
  }
  .tag {
    padding: 1px 6px;
    border: 1px solid var(--border-strong);
    border-radius: 4px;
    color: var(--muted);
    font-size: var(--fs-xs);
  }
  .tag.ok {
    border-color: transparent;
    background: var(--green-bg);
    color: var(--green);
  }
  .icon {
    display: grid;
    place-items: center;
    width: 28px;
    height: 28px;
    flex-shrink: 0;
    padding: 0;
    border: none;
    border-radius: 4px;
    background: none;
    color: var(--text);
    transition: background var(--ease), opacity var(--ease);
  }
  .icon:hover:not(:disabled) {
    background: var(--selected);
  }
  .icon:disabled {
    opacity: 0.3;
  }
  .link {
    padding: 0;
    border: none;
    background: none;
    color: var(--text-2);
    font: inherit;
    font-size: var(--fs-sm);
    text-decoration: underline;
    text-underline-offset: 3px;
    transition: color var(--ease);
  }
  .link:hover {
    color: var(--text);
  }
  .link.danger,
  .btn.danger {
    color: var(--red);
  }
  .downloads {
    display: flex;
    flex-direction: column;
    gap: var(--sp-4);
  }
  .downloads li {
    display: flex;
    flex-direction: column;
    gap: var(--sp-2);
  }
  .d-head {
    display: flex;
    align-items: center;
    gap: var(--sp-3);
  }
  .d-head .r-title {
    flex-grow: 1;
  }
  .progress {
    height: 4px;
    border-radius: 2px;
    background: var(--selected);
    overflow: hidden;
  }
  .progress span {
    display: block;
    height: 100%;
    background: var(--primary);
  }
  .lists {
    display: flex;
    gap: var(--sp-5);
    min-height: 100%;
  }
  aside {
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
    gap: var(--sp-1);
    width: 220px;
  }
  .list-item {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
    height: 34px;
    padding: 0 var(--sp-3);
    border: none;
    border-radius: var(--r-sm);
    background: none;
    color: var(--muted);
    font: inherit;
    text-align: left;
    transition: background var(--ease), color var(--ease);
  }
  .list-item span:first-child {
    flex-grow: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .list-item:hover {
    color: var(--text);
  }
  .list-item.on {
    background: var(--selected);
    color: var(--text);
  }
  .count {
    font-size: var(--fs-xs);
  }
  .new {
    display: flex;
    gap: var(--sp-2);
    margin-top: var(--sp-2);
  }
  .new .field {
    min-width: 0;
  }
  section {
    display: flex;
    flex-direction: column;
    flex-grow: 1;
    gap: var(--sp-3);
    min-width: 0;
  }
  .list-head {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
  }
  .name {
    font-weight: 600;
  }
</style>
