import isPromise from 'p-is-promise'

import CacheLoader from './cache-loader'

const seconds = 1000
const minutes = 60 * seconds

const ERR_UNKNOWNCOMMAND = 421

const ErrExtJwtUnsupported = new Error('EXTJWT unsupported on this server/gateway')

const UNSUPPORTED_TTL = 5 * minutes

export default class TokenManager {
    constructor() {
        this.unsupportedNetworks = new Map()
        this.requestToken = this.requestToken.bind(this) // ?!?!
        this.cacheLoader = new CacheLoader(this.requestToken, TokenManager.assertValid)
    }

    get(network) {
        if (this.unsupportedNetworks.has(network)) {
            if (new Date() - this.unsupportedNetworks.get(network) < UNSUPPORTED_TTL) {
                return false // don't retry EXTJWT on unsupported servers
            }
        }

        const maybePromise = this.cacheLoader.get(network)

        if (isPromise(maybePromise)) {
            const tokenRecordPromise = maybePromise
            return tokenRecordPromise
                .then(tokenRecord => tokenRecord.token)
                .catch(err => {
                    if (err === ErrExtJwtUnsupported) {
                        return false
                    }
                    throw err
                })
        }

        const tokenRecord = maybePromise
        return tokenRecord.token
    }

    async requestToken(network) {
        const thisTokenManager = this

        const respPromise = awaitMessage(
            network.ircClient,
            message => {
                if (message.command === String(ERR_UNKNOWNCOMMAND) && message.params[1].toUpperCase() === 'EXTJWT') {
                    throw ErrExtJwtUnsupported
                }

                return message.command.toUpperCase() === 'EXTJWT' && message.params[0] === '*'
            },
            { timeout: 10 * seconds }
        )

        network.ircClient.raw('EXTJWT')

        let resp
        try {
            resp = await respPromise
        } catch (err) {
            if (err === ErrExtJwtUnsupported) {
                const unsupportedAt = new Date()
                thisTokenManager.unsupportedNetworks.set(network, unsupportedAt)
                console.debug('Network does not support EXTJWT:', network)
            }
            throw err
        }

        const acquiredAt = new Date()
        const [target, token] = resp.params
        return { token, acquiredAt }
    }

    static assertValid(tokenRecord) {
        const { token, acquiredAt } = tokenRecord
        const now = new Date()
        if (now - acquiredAt > 15 * seconds) {
            throw new Error(`Stale token: ${(now - acquiredAt) / 1000} seconds age exceeds 15 second limit`)
        }
    }
}

function awaitMessage(ircClient, matcher, { timeout } = { timeout: undefined }) {
    const { connection } = ircClient
    return new Promise((resolve, reject) => {
        let timeoutHandle
        if (timeout) {
            timeoutHandle = setTimeout(() => {
                connection.removeListener('message', callback)
                reject(new Error('Timeout expired'))
            }, timeout)
        }

        const callback = message => {
            try {
                if (matcher(message)) {
                    connection.removeListener('message', callback)
                    if (timeoutHandle) clearTimeout(timeoutHandle)
                    resolve(message)
                }
            } catch (err) {
                connection.removeListener('message', callback)
                if (timeoutHandle) clearTimeout(timeoutHandle)
                reject(err)
            }
        }

        connection.on('message', callback)
    })
}
