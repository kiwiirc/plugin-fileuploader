export function shareCompletedUploadUrl(kiwiApi) {
    return function handleUploadSuccess(file, response) {
        // append filename to uploadURL
        const url = `${response.uploadURL}/${encodeURIComponent(file.meta.name)}`

        // emit a global kiwi event
        kiwiApi.emit('fileuploader.uploaded', { url, file })

        // send a message with the url of each successful upload
        const buffer = file.kiwiFileUploaderTargetBuffer
        buffer.say(`Uploaded file: ${url}`)
    }
}
