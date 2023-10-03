// document.addEventListener("DOMContentLoaded", function(){
//     var selectedTheme = localStorage.getItem("theme")
//     if (selectedTheme){
// 	setTheme(selectedTheme)
//     } else {
// 	if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
// 	    setTheme("dark")
// 	} else {
// 	    setTheme("light")
// 	}
//     }
// })

// window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', event => {
//     setTheme("system")
// });

// function onThemeChange(ev){
//     setTheme(ev.target.value)
// }

// function setTheme(themeName){
//     document.getElementById("theme").value = themeName
//     localStorage.setItem("theme", themeName)

//     if (themeName == "system") {
// 	if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches){
// 	    themeName = "dark"
// 	} else {
// 	    themeName = "light"
// 	}
//     }
//     document.documentElement.setAttribute("data-theme", themeName)
// }
