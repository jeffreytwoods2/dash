function toggleDropdown() {
    let body = document.querySelector("body")
    let dropdown = document.getElementById("dropdown")
    body.classList.toggle("body-dropdown-closed")
    dropdown.classList.toggle("dropdown-closed")
}

window.onload = async () => {
    if ('serviceWorker' in navigator) {
        navigator.serviceWorker.register("/static/js/sw.js")
    }
    
    let menu = document.getElementById("menu")

    menu.addEventListener("click", toggleDropdown)

    const CACHE_INFO = await fetch("http://localhost:4000/sw").then((res) => {
        return res.json()
    })
    console.log("CACHE_INFO: ", CACHE_INFO)
}