window.onload = () => {
    let menu = document.getElementById("menu-btn")
    let nav = document.getElementById("dropdown")

    menu.addEventListener("click", () => {
        nav.classList.toggle("menu-open")
    })
}