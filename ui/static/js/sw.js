const CACHE_INFO = fetch("http://localhost:4000/sw").then((res) => {
    return res.json()
})
// console.log("CACHE_INFO: ", CACHE_INFO)

self.addEventListener("install", async (e) => {
    await CACHE_INFO
    async function addFilesToCache() {
        const cache = await caches.open(CACHE_INFO.version)
        await cache.addAll(...CACHE_INFO.static_files)
    }

    try {
        e.waitUntil(addFilesToCache())
    } catch (err) {
        console.log(err)
    }
})