export namespace audio {
	
	export class Status {
	    available: boolean;
	    missing: string[];
	    hint: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.missing = source["missing"];
	        this.hint = source["hint"];
	    }
	}

}

export namespace claude {
	
	export class Alternative {
	    name: string;
	    whyNot: string;
	
	    static createFrom(source: any = {}) {
	        return new Alternative(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.whyNot = source["whyNot"];
	    }
	}
	export class Attention {
	    line: number;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new Attention(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.line = source["line"];
	        this.note = source["note"];
	    }
	}
	export class Risk {
	    title: string;
	    detail: string;
	    level: string;
	    node: string;
	
	    static createFrom(source: any = {}) {
	        return new Risk(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.detail = source["detail"];
	        this.level = source["level"];
	        this.node = source["node"];
	    }
	}
	export class ImpactAxis {
	    axis: string;
	    score: number;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new ImpactAxis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.axis = source["axis"];
	        this.score = source["score"];
	        this.note = source["note"];
	    }
	}
	export class Decision {
	    title: string;
	    chosen: string;
	    why: string;
	    gains: string[];
	    costs: string[];
	    alternatives: Alternative[];
	
	    static createFrom(source: any = {}) {
	        return new Decision(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.chosen = source["chosen"];
	        this.why = source["why"];
	        this.gains = source["gains"];
	        this.costs = source["costs"];
	        this.alternatives = this.convertValues(source["alternatives"], Alternative);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Step {
	    from: string;
	    to: string;
	    action: string;
	    note: string;
	    change: string;
	
	    static createFrom(source: any = {}) {
	        return new Step(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	        this.action = source["action"];
	        this.note = source["note"];
	        this.change = source["change"];
	    }
	}
	export class Flow {
	    title: string;
	    summary: string;
	    steps: Step[];
	
	    static createFrom(source: any = {}) {
	        return new Flow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.summary = source["summary"];
	        this.steps = this.convertValues(source["steps"], Step);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Edge {
	    from: string;
	    to: string;
	    label: string;
	    style: string;
	    change: string;
	
	    static createFrom(source: any = {}) {
	        return new Edge(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	        this.label = source["label"];
	        this.style = source["style"];
	        this.change = source["change"];
	    }
	}
	export class Weight {
	    additions: number;
	    deletions: number;
	
	    static createFrom(source: any = {}) {
	        return new Weight(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.additions = source["additions"];
	        this.deletions = source["deletions"];
	    }
	}
	export class Node {
	    id: string;
	    label: string;
	    kind: string;
	    change: string;
	    role: string;
	    files: string[];
	    weight: Weight;
	
	    static createFrom(source: any = {}) {
	        return new Node(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.kind = source["kind"];
	        this.change = source["change"];
	        this.role = source["role"];
	        this.files = source["files"];
	        this.weight = this.convertValues(source["weight"], Weight);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Blueprint {
	    headline: string;
	    kind: string;
	    summary: string;
	    before: string;
	    after: string;
	    nodes: Node[];
	    edges: Edge[];
	    flows: Flow[];
	    decisions: Decision[];
	    impact: ImpactAxis[];
	    risks: Risk[];
	    questions: string[];
	
	    static createFrom(source: any = {}) {
	        return new Blueprint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.headline = source["headline"];
	        this.kind = source["kind"];
	        this.summary = source["summary"];
	        this.before = source["before"];
	        this.after = source["after"];
	        this.nodes = this.convertValues(source["nodes"], Node);
	        this.edges = this.convertValues(source["edges"], Edge);
	        this.flows = this.convertValues(source["flows"], Flow);
	        this.decisions = this.convertValues(source["decisions"], Decision);
	        this.impact = this.convertValues(source["impact"], ImpactAxis);
	        this.risks = this.convertValues(source["risks"], Risk);
	        this.questions = source["questions"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	export class Reference {
	    path: string;
	    why: string;
	
	    static createFrom(source: any = {}) {
	        return new Reference(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.why = source["why"];
	    }
	}
	
	

}

export namespace config {
	
	export class RepoConfig {
	    owner: string;
	    name: string;
	    onlyReviewRequested: boolean;
	    preAnalyze: boolean;
	    promptOverride: string;
	
	    static createFrom(source: any = {}) {
	        return new RepoConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.owner = source["owner"];
	        this.name = source["name"];
	        this.onlyReviewRequested = source["onlyReviewRequested"];
	        this.preAnalyze = source["preAnalyze"];
	        this.promptOverride = source["promptOverride"];
	    }
	}
	export class Config {
	    pollIntervalMinutes: number;
	    repos: RepoConfig[];
	    reviewerPrompt: string;
	    model: string;
	    effort: string;
	    launchAtLogin: boolean;
	    autostartNoticeSeen: boolean;
	    skippedVersion: string;
	    notifyOwnPRs: boolean;
	    includeDrafts: boolean;
	    defaultMode: string;
	    architectPrompt: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pollIntervalMinutes = source["pollIntervalMinutes"];
	        this.repos = this.convertValues(source["repos"], RepoConfig);
	        this.reviewerPrompt = source["reviewerPrompt"];
	        this.model = source["model"];
	        this.effort = source["effort"];
	        this.launchAtLogin = source["launchAtLogin"];
	        this.autostartNoticeSeen = source["autostartNoticeSeen"];
	        this.skippedVersion = source["skippedVersion"];
	        this.notifyOwnPRs = source["notifyOwnPRs"];
	        this.includeDrafts = source["includeDrafts"];
	        this.defaultMode = source["defaultMode"];
	        this.architectPrompt = source["architectPrompt"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace gh {
	
	export class PRRef {
	    owner: string;
	    repo: string;
	    number: number;
	
	    static createFrom(source: any = {}) {
	        return new PRRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.owner = source["owner"];
	        this.repo = source["repo"];
	        this.number = source["number"];
	    }
	}
	export class OpenPR {
	    ref: PRRef;
	    title: string;
	    url: string;
	    author: string;
	    headSha: string;
	    isDraft: boolean;
	    createdAt: string;
	    reviewRequested: string[];
	
	    static createFrom(source: any = {}) {
	        return new OpenPR(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = this.convertValues(source["ref"], PRRef);
	        this.title = source["title"];
	        this.url = source["url"];
	        this.author = source["author"];
	        this.headSha = source["headSha"];
	        this.isDraft = source["isDraft"];
	        this.createdAt = source["createdAt"];
	        this.reviewRequested = source["reviewRequested"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Repo {
	    fullName: string;
	    private: boolean;
	    pushedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Repo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fullName = source["fullName"];
	        this.private = source["private"];
	        this.pushedAt = source["pushedAt"];
	    }
	}

}

export namespace main {
	
	export class DepsStatus {
	    ghError: string;
	    ghInstalled: boolean;
	    missingScopes: string[];
	    claudeError: string;
	
	    static createFrom(source: any = {}) {
	        return new DepsStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ghError = source["ghError"];
	        this.ghInstalled = source["ghInstalled"];
	        this.missingScopes = source["missingScopes"];
	        this.claudeError = source["claudeError"];
	    }
	}
	export class DraftInput {
	    id: string;
	    path: string;
	    startLine: number;
	    line: number;
	    body: string;
	
	    static createFrom(source: any = {}) {
	        return new DraftInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.path = source["path"];
	        this.startLine = source["startLine"];
	        this.line = source["line"];
	        this.body = source["body"];
	    }
	}
	export class FileEntry {
	    path: string;
	    status: string;
	    additions: number;
	    deletions: number;
	    severity: string;
	    why: string;
	    explanation: string;
	    attention: claude.Attention[];
	    references: claude.Reference[];
	
	    static createFrom(source: any = {}) {
	        return new FileEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.status = source["status"];
	        this.additions = source["additions"];
	        this.deletions = source["deletions"];
	        this.severity = source["severity"];
	        this.why = source["why"];
	        this.explanation = source["explanation"];
	        this.attention = this.convertValues(source["attention"], claude.Attention);
	        this.references = this.convertValues(source["references"], claude.Reference);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FileView {
	    path: string;
	    status: string;
	    content: string;
	    added: number[];
	    commentable: number[];
	    severity: string;
	    why: string;
	    explanation: string;
	    attention: claude.Attention[];
	    readOnly: boolean;
	    inPR: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.status = source["status"];
	        this.content = source["content"];
	        this.added = source["added"];
	        this.commentable = source["commentable"];
	        this.severity = source["severity"];
	        this.why = source["why"];
	        this.explanation = source["explanation"];
	        this.attention = this.convertValues(source["attention"], claude.Attention);
	        this.readOnly = source["readOnly"];
	        this.inPR = source["inPR"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SessionView {
	    ref: gh.PRRef;
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
	    skipped: review.Skipped[];
	    drafts: review.Draft[];
	    fromCache: boolean;
	    repoFiles: string[];
	    mode: string;
	    blueprint?: claude.Blueprint;
	
	    static createFrom(source: any = {}) {
	        return new SessionView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ref = this.convertValues(source["ref"], gh.PRRef);
	        this.title = source["title"];
	        this.url = source["url"];
	        this.author = source["author"];
	        this.headBranch = source["headBranch"];
	        this.baseBranch = source["baseBranch"];
	        this.additions = source["additions"];
	        this.deletions = source["deletions"];
	        this.totalFiles = source["totalFiles"];
	        this.summary = source["summary"];
	        this.files = this.convertValues(source["files"], FileEntry);
	        this.others = this.convertValues(source["others"], FileEntry);
	        this.skipped = this.convertValues(source["skipped"], review.Skipped);
	        this.drafts = this.convertValues(source["drafts"], review.Draft);
	        this.fromCache = source["fromCache"];
	        this.repoFiles = source["repoFiles"];
	        this.mode = source["mode"];
	        this.blueprint = this.convertValues(source["blueprint"], claude.Blueprint);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SettingsView {
	    config: config.Config;
	    defaultPrompt: string;
	    defaultArchitectPrompt: string;
	    configDir: string;
	
	    static createFrom(source: any = {}) {
	        return new SettingsView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config = this.convertValues(source["config"], config.Config);
	        this.defaultPrompt = source["defaultPrompt"];
	        this.defaultArchitectPrompt = source["defaultArchitectPrompt"];
	        this.configDir = source["configDir"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UpdateInfo {
	    current: string;
	    version: string;
	    notes: string;
	    pageUrl: string;
	    needsPassword: boolean;
	    automatic: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current = source["current"];
	        this.version = source["version"];
	        this.notes = source["notes"];
	        this.pageUrl = source["pageUrl"];
	        this.needsPassword = source["needsPassword"];
	        this.automatic = source["automatic"];
	    }
	}

}

export namespace music {
	
	export class Library {
	    dir: string;
	    tracks: string[];
	
	    static createFrom(source: any = {}) {
	        return new Library(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dir = source["dir"];
	        this.tracks = source["tracks"];
	    }
	}

}

export namespace review {
	
	export class Draft {
	    id: string;
	    path: string;
	    startLine: number;
	    line: number;
	    body: string;
	    inDiff: boolean;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Draft(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.path = source["path"];
	        this.startLine = source["startLine"];
	        this.line = source["line"];
	        this.body = source["body"];
	        this.inDiff = source["inDiff"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class Skipped {
	    path: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new Skipped(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.reason = source["reason"];
	    }
	}

}

export namespace watcher {
	
	export class Snapshot {
	    pending: gh.OpenPR[];
	    lastPoll: string;
	    lastError: string;
	    checking: boolean;
	    repos: number;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pending = this.convertValues(source["pending"], gh.OpenPR);
	        this.lastPoll = source["lastPoll"];
	        this.lastError = source["lastError"];
	        this.checking = source["checking"];
	        this.repos = source["repos"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

