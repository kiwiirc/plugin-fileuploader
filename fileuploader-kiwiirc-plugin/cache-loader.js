export default class CacheLoader {
    // cache = new Map()
    // loading = new Map()
    // loadFn
    // assertValid

    constructor(loadFn, assertValid) {
        this.cache = new Map()
        this.loading = new Map()
        this.loadFn = loadFn
        this.assertValid = assertValid
    }

    get(key) {
        if (!this.cache.has(key)) {
            return this.load(key)
        }
        const val = this.cache.get(key)
        try {
            this.assertValid(val)
            return val
        } catch (err) {
            console.warn(`Cached value failed validation: ${err.message}`)
            return this.load(key)
        }
    }

    async load(key) {
        if (this.loading.has(key)) {
            return this.loading.get(key)
        }

        const valPromise = this.loadFn(key)
        this.loading.set(key, valPromise)

        try {
            const val = await valPromise
            this.assertValid(val)
            this.cache.set(key, val)
            return val
        } finally {
            this.loading.delete(key)
        }
    }
}
