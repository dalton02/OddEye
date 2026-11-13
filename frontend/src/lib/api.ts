import * as Go from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";

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

export interface WatcherSnapshot {
  pending: OpenPR[];
  lastPoll: string;
  lastError: string;
  checking: boolean;
  repos: number;
}

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

export const prRefLabel = (r: PRRef): string => `${r.owner}/${r.repo}#${r.number}`;

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
  listMusic: (): Promise<MusicLibrary> => Go.ListMusic() as unknown as Promise<MusicLibrary>,
  chooseMusicFolder: (): Promise<MusicLibrary | null> => Go.ChooseMusicFolder() as unknown as Promise<MusicLibrary | null>,
  openMusicFolder: (): Promise<void> => Go.OpenMusicFolder(),
  audioStatus: (): Promise<AudioStatus> => Go.AudioStatus() as unknown as Promise<AudioStatus>,
  musicURL: (track: string): string => "/music/" + encodeURIComponent(track),
  onProgress: (cb: (p: Progress) => void): (() => void) => EventsOn("analysis:progress", cb),
  getSettings: (): Promise<SettingsView> => Go.GetSettings() as unknown as Promise<SettingsView>,
  saveSettings: (c: Config): Promise<Config> =>
    Go.SaveSettings(c as unknown as Parameters<typeof Go.SaveSettings>[0]) as unknown as Promise<Config>,
  listRepos: (): Promise<RepoOption[]> => Go.ListRepos() as unknown as Promise<RepoOption[]>,
  watcherState: (): Promise<WatcherSnapshot> => Go.WatcherState() as unknown as Promise<WatcherSnapshot>,
  checkNow: (): Promise<void> => Go.CheckNow(),
  authorizeGitHub: (): Promise<void> => Go.AuthorizeGitHub(),
  cancelAuthorize: (): Promise<void> => Go.CancelAuthorize(),
  onWatcher: (cb: (s: WatcherSnapshot) => void): (() => void) => EventsOn("watcher:update", cb),
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
