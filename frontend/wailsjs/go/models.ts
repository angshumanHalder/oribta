export namespace proxy {
	
	export class RewriteRule {
	    From: string;
	    To: string;
	
	    static createFrom(source: any = {}) {
	        return new RewriteRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.From = source["From"];
	        this.To = source["To"];
	    }
	}

}

