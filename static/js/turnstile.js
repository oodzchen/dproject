(function(){
    try{
	const cfSiteKey = document.querySelector("#cf-site-key").value
	turnstile.ready(function () {
	    turnstile.render('#human-verify', {
		sitekey: cfSiteKey,
		callback: function(token) {
		    console.log(`Challenge Success ${token}`);
		    window.location = "/register?cf_ts_resp=" + window.encodeURIComponent(token)
		},
	    });
	})
    }catch(e){
	console.error("clourdflare turnstile init failed", e)
    }
})()
