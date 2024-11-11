(function () {
  try {
    const cfSiteKey = document.querySelector("#cf-site-key").value;
    turnstile.ready(function () {
      turnstile.render("#human-verify", {
        sitekey: cfSiteKey,
        callback: function (token) {
          document.cookie =
            "cf_ts_resp=" + window.encodeURIComponent(token) + ";path=/";
          setTimeout(function () {
            window.location = "/register";
          }, 0);
        },
      });
    });
  } catch (e) {
    console.error("clourdflare turnstile init failed", e);
  }
})();
