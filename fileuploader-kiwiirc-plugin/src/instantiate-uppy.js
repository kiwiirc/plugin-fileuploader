import Uppy from '@uppy/core'
import Dashboard from '@uppy/dashboard'
import Tus from '@uppy/tus'
import Webcam from '@uppy/webcam'
import isPromise from 'p-is-promise'

import { KiB } from './constants/data-size'
import { getValidUploadTarget } from './utils/get-valid-upload-target'

export default function instantiateUppy({ kiwiApi, tokenManager, uploadFileButton }) {
    const uppy = Uppy({
        autoProceed: false,
        onBeforeFileAdded: (/* currentFile, files */) => {
            // throws if invalid, cancelling the file add
            getValidUploadTarget(kiwiApi)
        },
        onBeforeUpload: files => {
            const uniqNetworks = new Set(
                Object.values(files)
                    .map(file =>
                        file.kiwiFileUploaderTargetBuffer.getNetwork()
                    )
            )

            const tokens = new Map()
            for (const network of uniqNetworks) {
                tokens.set(network, tokenManager.get(network))
            }

            const tokenPromises = [...tokens.values()].filter(isPromise)
            if (tokenPromises.length > 0) {
                console.debug('Tokens were not available synchronously. Cancelling upload start and acquiring tokens.')

                // restart uploads once all needed tokens are acquired
                Promise.all(tokenPromises).then(() => {
                    console.debug('Token acquisition complete. Restarting upload.')
                    uppy.upload()
                }).catch(err => {
                    uppy.info({
                        message: 'Unhandled error acquiring EXTJWT tokens!',
                        details: err,
                    }, 'error', 5000)
                })

                // prevent uploads from starting now
                return false
            }

            // WARNING: don't use an immutable update pattern here!
            // if we return new objects, uppy will ignore our changes
            for (const fileObj of Object.values(files)) {
                const token = tokenManager.get(fileObj.kiwiFileUploaderTargetBuffer.getNetwork())
                if (token) { // token will be false if the server response was 'Unknown Command'
                    fileObj.meta['extjwt'] = token
                }
            }
        },
        restrictions: {
            maxFileSize: kiwiApi.state.setting('fileuploader.maxFileSize'),
        },
    })
        .use(Dashboard, {
            trigger: uploadFileButton,
            proudlyDisplayPoweredByUppy: false,
            closeModalOnClickOutside: true,
            note: kiwiApi.state.setting('fileuploader.note'),
        })
        .use(Webcam, { target: Dashboard })
        .use(Tus, {
            endpoint: kiwiApi.state.setting('fileuploader.server'),
            chunkSize: 512 * KiB,
        })
        .run()

    const dashboard = uppy.getPlugin('Dashboard')

    return { uppy, dashboard }
}
