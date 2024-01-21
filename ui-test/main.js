function toggleDropdown() {
    let body = document.querySelector("body")
    let dropdown = document.getElementById("dropdown")
    body.classList.toggle("body-dropdown-closed")
    dropdown.classList.toggle("dropdown-closed")
}

window.onload = () => {
    let menu = document.getElementById("menu")
    let dropdownLinks = document.querySelectorAll("#dropdown a")

    menu.addEventListener("click", toggleDropdown)
}