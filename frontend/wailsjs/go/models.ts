export namespace api {
	
	export class EntryDTO {
	    line_number: number;
	    channel: string;
	    timestamp: string;
	    sender: string;
	    display_name: string;
	    realm: string;
	    status_symbol: string;
	    message: string;
	    spans: highlight.Span[];
	    part_index: number;
	    part_total: number;
	    is_continuation: boolean;
	    has_continuation: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EntryDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.line_number = source["line_number"];
	        this.channel = source["channel"];
	        this.timestamp = source["timestamp"];
	        this.sender = source["sender"];
	        this.display_name = source["display_name"];
	        this.realm = source["realm"];
	        this.status_symbol = source["status_symbol"];
	        this.message = source["message"];
	        this.spans = this.convertValues(source["spans"], highlight.Span);
	        this.part_index = source["part_index"];
	        this.part_total = source["part_total"];
	        this.is_continuation = source["is_continuation"];
	        this.has_continuation = source["has_continuation"];
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
	export class LogSummary {
	    file_path: string;
	    file_name: string;
	    log_date: string;
	    message_count: number;
	    participants: string[];
	    channels: string[];
	    duration: string;
	    tags: string[];
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new LogSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_path = source["file_path"];
	        this.file_name = source["file_name"];
	        this.log_date = source["log_date"];
	        this.message_count = source["message_count"];
	        this.participants = source["participants"];
	        this.channels = source["channels"];
	        this.duration = source["duration"];
	        this.tags = source["tags"];
	        this.note = source["note"];
	    }
	}
	export class SearchResultDTO {
	    file_path: string;
	    file_name: string;
	    line_number: number;
	    channel: string;
	    sender: string;
	    snippet: string;
	    score: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_path = source["file_path"];
	        this.file_name = source["file_name"];
	        this.line_number = source["line_number"];
	        this.channel = source["channel"];
	        this.sender = source["sender"];
	        this.snippet = source["snippet"];
	        this.score = source["score"];
	    }
	}
	export class SearchResponse {
	    results: SearchResultDTO[];
	    truncated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SearchResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.results = this.convertValues(source["results"], SearchResultDTO);
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
	
	export class SetupState {
	    needs_setup: boolean;
	    config_exists: boolean;
	    default_log_dir: string;
	    default_log_dir_exists: boolean;
	    wizard_version: number;
	
	    static createFrom(source: any = {}) {
	        return new SetupState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.needs_setup = source["needs_setup"];
	        this.config_exists = source["config_exists"];
	        this.default_log_dir = source["default_log_dir"];
	        this.default_log_dir_exists = source["default_log_dir_exists"];
	        this.wizard_version = source["wizard_version"];
	    }
	}
	export class ThreadDTO {
	    sender: string;
	    channel: string;
	    lines: number[];
	    combined: string;
	    spans: highlight.Span[];
	    start_time: string;
	    end_time: string;
	
	    static createFrom(source: any = {}) {
	        return new ThreadDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sender = source["sender"];
	        this.channel = source["channel"];
	        this.lines = source["lines"];
	        this.combined = source["combined"];
	        this.spans = this.convertValues(source["spans"], highlight.Span);
	        this.start_time = source["start_time"];
	        this.end_time = source["end_time"];
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
	export class UpdateCheckResult {
	    status: string;
	    current_version: string;
	    latest_version: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.current_version = source["current_version"];
	        this.latest_version = source["latest_version"];
	    }
	}

}

export namespace config {
	
	export class Config {
	    log_directories: string[];
	    auto_detect_appdata: boolean;
	    language: string;
	    mention_names: string[];
	    roleplay_characters: string[];
	    markers: highlight.MarkerSet;
	    theme: string;
	    channel_filters: Record<string, boolean>;
	    check_updates_on_start: boolean;
	    setup_wizard_version: number;
	    colors: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.log_directories = source["log_directories"];
	        this.auto_detect_appdata = source["auto_detect_appdata"];
	        this.language = source["language"];
	        this.mention_names = source["mention_names"];
	        this.roleplay_characters = source["roleplay_characters"];
	        this.markers = this.convertValues(source["markers"], highlight.MarkerSet);
	        this.theme = source["theme"];
	        this.channel_filters = source["channel_filters"];
	        this.check_updates_on_start = source["check_updates_on_start"];
	        this.setup_wizard_version = source["setup_wizard_version"];
	        this.colors = source["colors"];
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

export namespace highlight {
	
	export class MarkerPair {
	    open: string;
	    close: string;
	
	    static createFrom(source: any = {}) {
	        return new MarkerPair(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.open = source["open"];
	        this.close = source["close"];
	    }
	}
	export class MarkerSet {
	    speech: MarkerPair[];
	    emote: MarkerPair[];
	    ooc: MarkerPair[];
	
	    static createFrom(source: any = {}) {
	        return new MarkerSet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.speech = this.convertValues(source["speech"], MarkerPair);
	        this.emote = this.convertValues(source["emote"], MarkerPair);
	        this.ooc = this.convertValues(source["ooc"], MarkerPair);
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
	export class Span {
	    type: string;
	    text: string;
	    start: number;
	    end: number;
	
	    static createFrom(source: any = {}) {
	        return new Span(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.text = source["text"];
	        this.start = source["start"];
	        this.end = source["end"];
	    }
	}

}

export namespace tags {
	
	export class FileTags {
	    file_name: string;
	    tags: string[];
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new FileTags(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_name = source["file_name"];
	        this.tags = source["tags"];
	        this.note = source["note"];
	    }
	}

}

