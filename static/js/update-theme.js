(function() {
    const themeColorMap = {
        light: "#E8F4FF",
        dark: "#2E3841"
    }

    var rawTheme = getRawTheme()
    setTheme(rawTheme)
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', event => {
        setTheme(rawTheme)
    });
    
    function getRawTheme() {
        return document.documentElement.getAttribute('data-raw-theme')
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
        document.querySelector('meta[name="theme-color"]').content = themeColorMap[themeName]
    }
})()
