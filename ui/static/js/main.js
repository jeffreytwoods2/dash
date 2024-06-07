async function toggleDropdown() {
    let body = document.querySelector("body")
    let dropdown = document.getElementById("dropdown")
    body.classList.toggle("body-dropdown-closed")
    dropdown.classList.toggle("dropdown-closed")
}

window.onload = async () => {
    // Service worker not needed yet, but was fun to build
    // if ('serviceWorker' in navigator) {
    //     navigator.serviceWorker.register("https://dash.jwoods.dev/serviceworker")
    // }
    
    let menu = document.getElementById("menu")

    menu.addEventListener("click", toggleDropdown)
}