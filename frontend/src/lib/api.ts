import * as Go from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import type { Track, TorrentFile } from "./music";

export type { Track, TorrentFile } from "./music";

export type Severity = "critical" | "high" | "medium" | "";

export interface Attention {
  line: number;
  note: string;
}

export interface FileEntry {
  path: string;
  status: string;
  additions: number;
  deletions: number;
  severity: Severity;
  why: string;
  explanation: string;
  attention: Attention[];
  references: Reference[];
}

export interface Reference {
  path: string;
  why: string;
}

export interface Skipped {
  path: string;
  reason: string;
}

export interface Draft {
  id: string;
  path: string;
  startLine: number;
  line: number;
  body: string;
  inDiff: boolean;
}

export type Mode = "code" | "architect";

export const modeLabel: Record<Mode, string> = { code: "Código", architect: "Arquiteto" };

export type NodeKind = "actor" | "interface" | "service" | "worker" | "store" | "queue" | "external";
export type Change = "new" | "changed" | "removed" | "context";
export type EdgeStyle = "sync" | "async" | "data";
export type ImpactAxisName = "simplicity" | "reliability" | "performance" | "maintainability" | "decoupling" | "operability";

export interface BlueprintNode {
  id: string;
  label: string;
  kind: NodeKind;
  change: Change;
  role: string;
  files: string[];
  weight: { additions: number; deletions: number };
}

export interface BlueprintEdge {
  from: string;
  to: string;
  label: string;
  style: EdgeStyle;
  change: Change;
}

export interface FlowStep {
  from: string;
  to: string;
  action: string;
  note: string;
  change: Change;
}

export interface Flow {
  title: string;
  summary: string;
  steps: FlowStep[];
}

export interface Decision {
  title: string;
  chosen: string;
  why: string;
  gains: string[];
  costs: string[];
  alternatives: { name: string; whyNot: string }[];
}

export interface ImpactAxis {
  axis: ImpactAxisName;
  score: number;
  note: string;
}

export interface Risk {
  title: string;
  detail: string;
  level: "high" | "medium" | "low";
  node: string;
}

/** Leitura de arquitetura da PR (modo Arquiteto). Nenhum texto cita código; os arquivos ficam em nodes[].files. */
export interface Blueprint {
  headline: string;
  kind: string;
  summary: string;
  before: string;
  after: string;
  nodes: BlueprintNode[];
  edges: BlueprintEdge[];
  flows: Flow[];
  decisions: Decision[];
  impact: ImpactAxis[];
  risks: Risk[];
  questions: string[];
}

export interface Session {
  ref: { owner: string; repo: string; number: number };
  title: string;
  url: string;
  author: string;
  headBranch: string;
  baseBranch: string;
  additions: number;
  deletions: number;
  totalFiles: number;
  summary: string;
  files: FileEntry[];
  others: FileEntry[];
  skipped: Skipped[];
  drafts: Draft[];
  fromCache: boolean;
  repoFiles: string[];
  mode: Mode;
  blueprint: Blueprint | null;
}

export interface FileView {
  path: string;
  status: string;
  content: string;
  added: number[];
  commentable: number[];
  severity: Severity;
  why: string;
  explanation: string;
  attention: Attention[];
  readOnly: boolean;
  inPR: boolean;
}

export interface DraftInput {
  id: string;
  path: string;
  startLine: number;
  line: number;
  body: string;
}

export interface Progress {
  stage: "fetching" | "starting" | "thinking" | "writing" | "done";
  message: string;
  files: number;
}

export interface MusicLibrary {
  dir: string;
  tracks: string[];
}

/** Se o webview consegue tocar som; no Linux depende dos plugins do GStreamer instalados. */
export interface AudioStatus {
  available: boolean;
  missing: string[];
  hint: string;
}

export interface ArchiveItem {
  id: string;
  title: string;
  creator: string;
  date: string;
  downloads: number;
  license: string;
}

export interface ArchiveResult {
  total: number;
  items: ArchiveItem[];
}

export interface TorrentInfo {
  infoHash: string;
  name: string;
  title: string;
  size: number;
  done: number;
  wanted: number;
  complete: number;
  rate: number;
  peers: number;
}

export interface Playlist {
  id: string;
  name: string;
  tracks: Track[];
}

export type ReviewEvent = "COMMENT" | "APPROVE" | "REQUEST_CHANGES";

export interface Deps {
  ghError: string;
  ghInstalled: boolean;
  missingScopes: string[];
  claudeError: string;
}

export interface PRRef {
  owner: string;
  repo: string;
  number: number;
}

export interface OpenPR {
  ref: PRRef;
  title: string;
  url: string;
  author: string;
  headSha: string;
  isDraft: boolean;
  createdAt: string;
  reviewRequested: string[];
}

/** PRs abertas de um repositório vigiado; na tela inicial cada repo é um projeto */
export interface ProjectPRs {
  repo: string;
  prs: OpenPR[];
}

export const prRefLabel = (r: PRRef): string => `${r.owner}/${r.repo}#${r.number}`;

export interface RepoConfig {
  owner: string;
  name: string;
  onlyReviewRequested: boolean;
  preAnalyze: boolean;
  promptOverride: string;
}

export interface Config {
  pollIntervalMinutes: number;
  repos: RepoConfig[];
  reviewerPrompt: string;
  model: string;
  /** esforço de raciocínio do Claude: "" (padrão do Claude Code), low, medium, high, xhigh ou max */
  effort: string;
  launchAtLogin: boolean;
  autostartNoticeSeen: boolean;
  skippedVersion: string;
  notifyOwnPRs: boolean;
  includeDrafts: boolean;
  defaultMode: Mode;
  architectPrompt: string;
}

export interface UpdateInfo {
  current: string;
  version: string;
  notes: string;
  pageUrl: string;
  needsPassword: boolean;
  automatic: boolean;
}

export type UpdateStage = "downloading" | "installing" | "restarting" | "error";

export interface UpdateProgress {
  stage: UpdateStage;
  done: number;
  total: number;
  error?: string;
  manual?: boolean;
}

export interface SettingsView {
  config: Config;
  defaultPrompt: string;
  defaultArchitectPrompt: string;
  configDir: string;
}

export interface RepoOption {
  fullName: string;
  private: boolean;
  pushedAt: string;
}

export interface DeviceCode {
  code: string;
  url: string;
}

/* As bindings geradas pelo Wails usam classes com conversores; aqui só precisamos da forma JSON. */
export const api = {
  checkDeps: (): Promise<Deps> => Go.CheckDeps() as unknown as Promise<Deps>,
  loadPR: (url: string, mode: Mode, force: boolean): Promise<Session> =>
    Go.LoadPR(url, mode, force) as unknown as Promise<Session>,
  cancel: (): Promise<void> => Go.CancelAnalysis(),
  openFile: (path: string): Promise<FileView> => Go.OpenFile(path) as unknown as Promise<FileView>,
  saveDraft: (d: DraftInput): Promise<Draft[]> => Go.SaveDraft(d) as unknown as Promise<Draft[]>,
  removeDraft: (id: string): Promise<Draft[]> => Go.RemoveDraft(id) as unknown as Promise<Draft[]>,
  submitReview: (event: ReviewEvent, body: string): Promise<string> => Go.SubmitReview(event, body),
  openURL: (url: string): Promise<void> => Go.OpenURL(url),
  /** descrição original da PR aberta, em HTML renderizado pelo GitHub (URLs de mídia expiram: busque a cada abertura) */
  prDescription: (): Promise<string> => Go.PRDescription(),
  listMusic: (): Promise<MusicLibrary> => Go.ListMusic() as unknown as Promise<MusicLibrary>,
  chooseMusicFolder: (): Promise<MusicLibrary | null> => Go.ChooseMusicFolder() as unknown as Promise<MusicLibrary | null>,
  openMusicFolder: (): Promise<void> => Go.OpenMusicFolder(),
  audioStatus: (): Promise<AudioStatus> => Go.AudioStatus() as unknown as Promise<AudioStatus>,
  musicURL: (track: string): string => "/music/" + encodeURIComponent(track),
  searchArchive: (query: string, page: number): Promise<ArchiveResult> => Go.SearchArchive(query, page) as unknown as Promise<ArchiveResult>,
  addArchiveItem: (id: string, title: string): Promise<TorrentInfo> => Go.AddArchiveItem(id, title) as unknown as Promise<TorrentInfo>,
  addTorrent: (src: string): Promise<TorrentInfo> => Go.AddTorrent(src) as unknown as Promise<TorrentInfo>,
  chooseTorrentFile: (): Promise<TorrentInfo | null> => Go.ChooseTorrentFile() as unknown as Promise<TorrentInfo | null>,
  listTorrents: (): Promise<TorrentInfo[]> => Go.ListTorrents() as unknown as Promise<TorrentInfo[]>,
  torrentFiles: (hash: string): Promise<TorrentFile[]> => Go.TorrentFiles(hash) as unknown as Promise<TorrentFile[]>,
  downloadTrack: (hash: string, path: string, urgent: boolean): Promise<TorrentFile> =>
    Go.DownloadTrack(hash, path, urgent) as unknown as Promise<TorrentFile>,
  removeTorrent: (hash: string, deleteFiles: boolean): Promise<void> => Go.RemoveTorrent(hash, deleteFiles),
  onTorrentProgress: (cb: (list: TorrentInfo[]) => void): (() => void) => EventsOn("torrent:progress", cb),
  listPlaylists: (): Promise<Playlist[]> => Go.ListPlaylists() as unknown as Promise<Playlist[]>,
  createPlaylist: (name: string): Promise<Playlist> => Go.CreatePlaylist(name) as unknown as Promise<Playlist>,
  renamePlaylist: (id: string, name: string): Promise<void> => Go.RenamePlaylist(id, name),
  deletePlaylist: (id: string): Promise<void> => Go.DeletePlaylist(id),
  addToPlaylist: (id: string, tracks: Track[]): Promise<Playlist> =>
    Go.AddToPlaylist(id, tracks as unknown as Parameters<typeof Go.AddToPlaylist>[1]) as unknown as Promise<Playlist>,
  removeFromPlaylist: (id: string, trackID: string): Promise<Playlist> => Go.RemoveFromPlaylist(id, trackID) as unknown as Promise<Playlist>,
  movePlaylistTrack: (id: string, from: number, to: number): Promise<Playlist> =>
    Go.MovePlaylistTrack(id, from, to) as unknown as Promise<Playlist>,
  onProgress: (cb: (p: Progress) => void): (() => void) => EventsOn("analysis:progress", cb),
  getSettings: (): Promise<SettingsView> => Go.GetSettings() as unknown as Promise<SettingsView>,
  saveSettings: (c: Config): Promise<Config> =>
    Go.SaveSettings(c as unknown as Parameters<typeof Go.SaveSettings>[0]) as unknown as Promise<Config>,
  listRepos: (): Promise<RepoOption[]> => Go.ListRepos() as unknown as Promise<RepoOption[]>,
  openPRsByProject: (): Promise<ProjectPRs[]> => Go.OpenPRsByProject() as unknown as Promise<ProjectPRs[]>,
  authorizeGitHub: (): Promise<void> => Go.AuthorizeGitHub(),
  cancelAuthorize: (): Promise<void> => Go.CancelAuthorize(),
  onOpenPR: (cb: (ref: string) => void): (() => void) => EventsOn("pr:open", cb),
  onShowSettings: (cb: () => void): (() => void) => EventsOn("ui:settings", cb),
  onDeviceCode: (cb: (d: DeviceCode) => void): (() => void) => EventsOn("gh:device-code", cb),

  version: (): Promise<string> => Go.Version(),
  checkUpdate: (): Promise<UpdateInfo | null> => Go.CheckUpdate() as unknown as Promise<UpdateInfo | null>,
  pendingUpdate: (): Promise<UpdateInfo | null> => Go.PendingUpdate() as unknown as Promise<UpdateInfo | null>,
  installUpdate: (): Promise<void> => Go.InstallUpdate(),
  skipUpdate: (version: string): Promise<void> => Go.SkipUpdate(version),
  openReleasePage: (): Promise<void> => Go.OpenReleasePage(),
  onUpdateAvailable: (cb: (u: UpdateInfo) => void): (() => void) => EventsOn("update:available", cb),
  onUpdateProgress: (cb: (p: UpdateProgress) => void): (() => void) => EventsOn("update:progress", cb)
};

export function errorMessage(e: unknown): string {
  if (typeof e === "string") return e;
  if (e instanceof Error) return e.message;
  return String(e);
}

export const severityLabel: Record<string, string> = {
  critical: "Crítico",
  high: "Alto",
  medium: "Médio",
  "": "Fora da seleção"
};
