const seconds = 1000;

export default class TokenManager {
    constructor(kiwiApi) {
        this.kiwiApi = kiwiApi;
        this.cache = new Map();
    }

    get(network) {
        const cachedTokenOrPromise = this.cache.get(network);
        if (cachedTokenOrPromise) {
            const { tokenOrPromise, acquiredAt } = cachedTokenOrPromise;
            if (tokenOrPromise instanceof Promise) {
                // Token not yet resolved
                return tokenOrPromise;
            }

            // Check if the token is still valid
            const now = new Date();
            if (now - acquiredAt < 15 * seconds) {
                // Token valid
                return tokenOrPromise;
            }

            // Token expired
            this.cache.delete(network);
        }

        // Cache does not have a valid entry for this network
        const tokenPromise = this.getExtjwtToken(network);

        // Store the new token promise within the cache
        this.cache.set(network, {
            tokenOrPromise: tokenPromise,
            acquiredAt: new Date(),
        });

        return tokenPromise;
    }

    getExtjwtToken(network) {
        return new Promise((resolve) => {
            let fullToken = '';

            const callback = (command, event, eventNetwork) => {
                if (network !== eventNetwork) {
                    // Not a token for this network
                    return;
                }

                event.handled = true;
                fullToken += event.params[event.params.length - 1];
                if (event.params.length === 4) {
                    // Incomplete token, it will continue in the next message
                    return;
                }

                this.kiwiApi.off('irc.raw.EXTJWT', callback);

                // Replace the promise with the real token
                const cachedPromise = this.cache.get(network);
                cachedPromise.tokenOrPromise = fullToken;

                // Resolve the promise
                resolve();
            };

            this.kiwiApi.on('irc.raw.EXTJWT', callback);
            network.ircClient.raw('EXTJWT', '*');
        });
    }
}
