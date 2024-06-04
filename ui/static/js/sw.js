let version
let files

self.addEventListener("install", async (e) => {
    self.skipWaiting()
    const res = await fetch("https://dash.jwoods.dev/sw")
    const cacheInfo = await res.json()
    version = cacheInfo.version
    files = [...cacheInfo.static_files]

    try {
        const cache = await caches.open(version)
        await cache.addAll(files)
    } catch (err) {
        console.log(err)
    }
})

self.addEventListener('activate', (e) => {
	async function deleteOldCaches() {
		for (const key of await caches.keys()) {
			if (key !== version) {
                await caches.delete(key)
            }
		}
	}

	e.waitUntil(deleteOldCaches());
    return self.clients.claim()
});

self.addEventListener('fetch', (e) => {
    if (e.request.method !== 'GET') {
        return
    }

    async function respond() {
        const url = new URL(e.request.url)
        const cache = await caches.open(version)

        if (files.includes(url.pathname)) {
            const response = await cache.match(url.pathname)
            if (response) {
                return response
            }
        }

        // Keeping this boilerplate in case I want to cache certain
        // responses in the future. For now it seems completely useless
        // since server data changes by the second

        // try {
        //     const response = await fetch(e.request)

        //     if (!(response instanceof Response)) {
        //         throw new Error('invalid response from fetch')
        //     }

        //     if (response.status === 200) {
		// 		cache.put(e.request, response.clone())
		// 	}
            
        //     return response
        // } catch (err) {
        //     const response = await cache.match(e.request)

        //     if (response) {
		// 		return response
		// 	}

        //     throw err
        // }
    }

    e.respondWith(respond())
})