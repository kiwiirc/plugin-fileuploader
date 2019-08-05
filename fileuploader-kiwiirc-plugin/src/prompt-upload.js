import instantiateUppy from "./instantiate-uppy"

export function createPromptUpload({ kiwiApi, tokenManager }) {
    return function promptUpload(uppyOptions = {}) {
        const opts = {
            kiwiApi,
            tokenManager,
            ...uppyOptions,
        }

        return new Promise((resolve, reject) => {
            const { uppy, dashboard } = instantiateUppy(opts)

            uppy.on('file-added', file => {
                // needed for acquireExtjwtBeforeUpload
                file.kiwiFileUploaderTargetBuffer =
                    opts.kiwiApi.state.getActiveNetwork().serverBuffer()
            })

            uppy.on('complete', event => {
                resolve(event)
                dashboard.closeModal()
            })

            uppy.on('dashboard:modal-closed', () => {
                reject('Upload dialog closed')
                uppy.close()
            })

            dashboard.openModal()
        })
    }
}
