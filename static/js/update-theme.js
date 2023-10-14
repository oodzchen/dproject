(function(){
    var primaryTheme = getPrimaryTheme()
    if (primaryTheme == 'system') {
	setTheme(primaryTheme)
	window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', event => {
            setTheme(primaryTheme)
	});
    }
    function getPrimaryTheme() {
	return document.documentElement.getAttribute('data-theme')
    }

    function setTheme(themeName) {
	if (themeName == "system") {
	    if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
		themeName = "dark"
	    } else {
		themeName = "light"
	    }
	}
	document.documentElement.setAttribute("data-theme", themeName)
    }
})()
