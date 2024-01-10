document.onload(() => {
    const menuBtn = document.getElementById("menu-btn")
    const nav = document.getElementById("dropdown")

    menuBtn.addEventListener("click", () => nav.classList.toggle("menu-toggle"))
})