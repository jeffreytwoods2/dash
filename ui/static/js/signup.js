function validateBedrock(platform) {
    let gamertag = document.getElementById("gamertag")
    let firstChar = gamertag.value.charAt(0)
    let gamertagField = document.getElementById("gamertag-field")

    if (document.getElementById("gamertag-info") !== null) {
        document.getElementById("gamertag-info").remove()
    }

    const info = document.createElement("label")
    info.setAttribute("id", "gamertag-info")

    if (platform == "bedrock" && firstChar !== ".") {
        gamertag.value = "." + gamertag.value
        info.innerText = "A dot was prepended to your username for server purposes."
    } else if (platform == "java" && firstChar == ".") {
        gamertag.value = gamertag.value.slice(1)
        info.innerText = "The leading dot was removed from your username for server purposes."
    }

    gamertagField.insertAdjacentElement("afterend", info)
}