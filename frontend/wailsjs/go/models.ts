export namespace main {
	
	export class FileItem {
	    id: string;
	    originalName: string;
	    extension: string;
	    path: string;
	    isDir: boolean;
	    newName: string;
	    status: string;
	    errorMsg: string;
	    modTime: number;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new FileItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.originalName = source["originalName"];
	        this.extension = source["extension"];
	        this.path = source["path"];
	        this.isDir = source["isDir"];
	        this.newName = source["newName"];
	        this.status = source["status"];
	        this.errorMsg = source["errorMsg"];
	        this.modTime = source["modTime"];
	        this.size = source["size"];
	    }
	}
	export class RenameRule {
	    type: string;
	    params: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new RenameRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.params = source["params"];
	    }
	}
	export class ScriptItem {
	    path: string;
	    name: string;
	    desc: string;
	
	    static createFrom(source: any = {}) {
	        return new ScriptItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.desc = source["desc"];
	    }
	}

}

