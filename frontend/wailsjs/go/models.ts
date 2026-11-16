export namespace archive {
	
	export class Item {
	    id: string;
	    title: string;
	    creator: string;
	    date: string;
	    downloads: number;
	    license: string;
	
	    static createFrom(source: any = {}) {
	        return new Item(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.creator = source["creator"];
	        this.date = source["date"];
	        this.downloads = source["downloads"];
	        this.license = source["license"];
	    }
	}
	export class Result {
	    total: number;
	    items: Item[];
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.items = this.convertValues(source["items"], Item);
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
	
	export class BoardPrefs {
	    owner: string;
	    number: number;
	    lastRepo: string;
	    hidden: string[];
	
	    static createFrom(source: any = {}) {
	        return new BoardPrefs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.owner = source["owner"];
	        this.number = source["number"];
	        this.lastRepo = source["lastRepo"];
	        this.hidden = source["hidden"];
	    }
	}
	export class CodingTab {
	    id: string;
	    dir: string;
	    kind?: string;
	
	    static createFrom(source: any = {}) {
	        return new CodingTab(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.dir = source["dir"];
	        this.kind = source["kind"];
	    }
	}
	export class CodingPrefs {
	    tabs: CodingTab[];
	    active: string;
	
	    static createFrom(source: any = {}) {
	        return new CodingPrefs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tabs = this.convertValues(source["tabs"], CodingTab);
	        this.active = source["active"];
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
	
	export class BoardCard {
	    itemId: string;
	    kind: string;
	    contentId: string;
	    number: number;
	    title: string;
	    url: string;
	    state: string;
	    repo: string;
	    assignees: string[];
	    statusId: string;
	    typeId: string;
	    subTotal: number;
	    subDone: number;
	
	    static createFrom(source: any = {}) {
	        return new BoardCard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.itemId = source["itemId"];
	        this.kind = source["kind"];
	        this.contentId = source["contentId"];
	        this.number = source["number"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.state = source["state"];
	        this.repo = source["repo"];
	        this.assignees = source["assignees"];
	        this.statusId = source["statusId"];
	        this.typeId = source["typeId"];
	        this.subTotal = source["subTotal"];
	        this.subDone = source["subDone"];
	    }
	}
	export class BoardOption {
	    id: string;
	    name: string;
	    description: string;
	    color: string;
	
	    static createFrom(source: any = {}) {
	        return new BoardOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.color = source["color"];
	    }
	}
	export class BoardField {
	    id: string;
	    name: string;
	    options: BoardOption[];
	
	    static createFrom(source: any = {}) {
	        return new BoardField(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.options = this.convertValues(source["options"], BoardOption);
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
	export class Board {
	    projectId: string;
	    owner: string;
	    number: number;
	    title: string;
	    url: string;
	    public: boolean;
	    status: BoardField;
	    type?: BoardField;
	    cards: BoardCard[];
	    repos: string[];
	    truncated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Board(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.owner = source["owner"];
	        this.number = source["number"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.public = source["public"];
	        this.status = this.convertValues(source["status"], BoardField);
	        this.type = this.convertValues(source["type"], BoardField);
	        this.cards = this.convertValues(source["cards"], BoardCard);
	        this.repos = source["repos"];
	        this.truncated = source["truncated"];
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
	
	
	
	export class BranchRef {
	    name: string;
	    oid: string;
	
	    static createFrom(source: any = {}) {
	        return new BranchRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.oid = source["oid"];
	    }
	}
	export class CreatedCard {
	    card: BoardCard;
	    warning: string;
	
	    static createFrom(source: any = {}) {
	        return new CreatedCard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.card = this.convertValues(source["card"], BoardCard);
	        this.warning = source["warning"];
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
	export class SubIssue {
	    id: string;
	    number: number;
	    title: string;
	    state: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new SubIssue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.number = source["number"];
	        this.title = source["title"];
	        this.state = source["state"];
	        this.url = source["url"];
	    }
	}
	export class IssueDetail {
	    id: string;
	    number: number;
	    title: string;
	    body: string;
	    url: string;
	    state: string;
	    repo: string;
	    repoId: string;
	    defaultBranch: string;
	    assignees: string[];
	    parent?: SubIssue;
	    subIssues: SubIssue[];
	    linked: string[];
	    branches: BranchRef[];
	
	    static createFrom(source: any = {}) {
	        return new IssueDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.number = source["number"];
	        this.title = source["title"];
	        this.body = source["body"];
	        this.url = source["url"];
	        this.state = source["state"];
	        this.repo = source["repo"];
	        this.repoId = source["repoId"];
	        this.defaultBranch = source["defaultBranch"];
	        this.assignees = source["assignees"];
	        this.parent = this.convertValues(source["parent"], SubIssue);
	        this.subIssues = this.convertValues(source["subIssues"], SubIssue);
	        this.linked = source["linked"];
	        this.branches = this.convertValues(source["branches"], BranchRef);
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
	export class NewCard {
	    projectId: string;
	    repo: string;
	    title: string;
	    body: string;
	    assignMe: boolean;
	    statusFieldId: string;
	    statusOptionId: string;
	    typeFieldId: string;
	    typeOptionId: string;
	
	    static createFrom(source: any = {}) {
	        return new NewCard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.repo = source["repo"];
	        this.title = source["title"];
	        this.body = source["body"];
	        this.assignMe = source["assignMe"];
	        this.statusFieldId = source["statusFieldId"];
	        this.statusOptionId = source["statusOptionId"];
	        this.typeFieldId = source["typeFieldId"];
	        this.typeOptionId = source["typeOptionId"];
	    }
	}
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
	
	export class ProjectSummary {
	    id: string;
	    number: number;
	    title: string;
	    url: string;
	    public: boolean;
	    updatedAt: string;
	    items: number;
	
	    static createFrom(source: any = {}) {
	        return new ProjectSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.number = source["number"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.public = source["public"];
	        this.updatedAt = source["updatedAt"];
	        this.items = source["items"];
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
	export class ProjectPRs {
	    repo: string;
	    prs: gh.OpenPR[];
	
	    static createFrom(source: any = {}) {
	        return new ProjectPRs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repo = source["repo"];
	        this.prs = this.convertValues(source["prs"], gh.OpenPR);
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

export namespace playlist {
	
	export class Track {
	    id: string;
	    title: string;
	    artist: string;
	    source: string;
	    name?: string;
	    infoHash?: string;
	    path?: string;
	
	    static createFrom(source: any = {}) {
	        return new Track(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.source = source["source"];
	        this.name = source["name"];
	        this.infoHash = source["infoHash"];
	        this.path = source["path"];
	    }
	}
	export class Playlist {
	    id: string;
	    name: string;
	    tracks: Track[];
	
	    static createFrom(source: any = {}) {
	        return new Playlist(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.tracks = this.convertValues(source["tracks"], Track);
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

export namespace term {
	
	export class Snapshot {
	    seq: number;
	    scrollback: string;
	    started: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seq = source["seq"];
	        this.scrollback = source["scrollback"];
	        this.started = source["started"];
	    }
	}

}

export namespace torrent {
	
	export class File {
	    path: string;
	    size: number;
	    done: number;
	    complete: boolean;
	    wanted: boolean;
	
	    static createFrom(source: any = {}) {
	        return new File(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.size = source["size"];
	        this.done = source["done"];
	        this.complete = source["complete"];
	        this.wanted = source["wanted"];
	    }
	}
	export class Info {
	    infoHash: string;
	    name: string;
	    title: string;
	    size: number;
	    done: number;
	    wanted: number;
	    complete: number;
	    rate: number;
	    peers: number;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.infoHash = source["infoHash"];
	        this.name = source["name"];
	        this.title = source["title"];
	        this.size = source["size"];
	        this.done = source["done"];
	        this.wanted = source["wanted"];
	        this.complete = source["complete"];
	        this.rate = source["rate"];
	        this.peers = source["peers"];
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

export namespace workspace {
	
	export class Change {
	    path: string;
	    status: string;
	    added: number;
	    removed: number;
	
	    static createFrom(source: any = {}) {
	        return new Change(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.status = source["status"];
	        this.added = source["added"];
	        this.removed = source["removed"];
	    }
	}
	export class Changes {
	    isRepo: boolean;
	    branch: string;
	    files: Change[];
	
	    static createFrom(source: any = {}) {
	        return new Changes(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isRepo = source["isRepo"];
	        this.branch = source["branch"];
	        this.files = this.convertValues(source["files"], Change);
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
	export class Entry {
	    name: string;
	    path: string;
	    dir: boolean;
	    ignored: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.dir = source["dir"];
	        this.ignored = source["ignored"];
	    }
	}
	export class FileContent {
	    content: string;
	    binary: boolean;
	    tooBig: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileContent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content = source["content"];
	        this.binary = source["binary"];
	        this.tooBig = source["tooBig"];
	    }
	}
	export class FileDiff {
	    original: string;
	    modified: string;
	    binary: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileDiff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.original = source["original"];
	        this.modified = source["modified"];
	        this.binary = source["binary"];
	    }
	}

}

