self.addEventListener("install", (e) => {
    async function addFilesToCache() {
        const res = await fetch("https://dash.jwoods.dev/sw")
        const cacheInfo = await res.json()
        const cache = await caches.open(cacheInfo.version)
        await cache.addAll(cacheInfo.static_files)
    }

    try {
        e.waitUntil(addFilesToCache())
    } catch (err) {
        console.log(err)
    }
})

self.addEventListener('activate', (e) => {
	async function deleteOldCaches() {
        const res = await fetch("https://dash.jwoods.dev/sw")
        const cacheInfo = await res.json()
		for (const key of await caches.keys()) {
			if (key !== cacheInfo.version) await caches.delete(key);
		}
	}

	e.waitUntil(deleteOldCaches());
});