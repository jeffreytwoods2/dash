window.onload = () => {
    let menu = document.getElementById("menu-btn")
    let nav = document.getElementById("dropdown")
    let main = document.querySelector("main")

    menu.addEventListener("click", () => {
        nav.classList.toggle("menu-open")
        main.classList.toggle("blur")
    })
}