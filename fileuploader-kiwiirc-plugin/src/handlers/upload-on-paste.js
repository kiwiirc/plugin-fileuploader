import { getValidUploadTarget } from '../utils/get-valid-upload-target'
import { numLines } from '../utils/num-lines'

export function uploadOnPaste(kiwiApi, uppy, dashboard) {
    return function handleBufferPaste(event) {
        // swallow error and ignore paste if no valid buffer to share to
        try {
            getValidUploadTarget(kiwiApi)
        } catch (err) {
            return
        }

        // IE 11 puts the clipboardData on the window
        const cbData = event.clipboardData || window.clipboardData

        if (!cbData) {
            return
        }

        // detect large text pastes, offer to create a file upload instead
        const text = cbData.getData('text')
        if (text) {
            const promptDisabled = kiwiApi.state.setting('fileuploader.textPasteNeverPrompt')
            if (promptDisabled) {
                return
            }
            const minLines = kiwiApi.state.setting('fileuploader.textPastePromptMinimumLines')
            const network = kiwiApi.state.getActiveNetwork()
            const networkMaxLineLen =
                network.ircClient.options.message_max_length
            if (text.length > networkMaxLineLen || numLines(text) >= minLines) {
                const msg =
                    'You pasted a lot of text.\nWould you like to upload as a file instead?'
                if (window.confirm(msg)) {
                    // stop IrcInput from ingesting the pasted text
                    event.preventDefault()
                    event.stopPropagation()

                    // only if there are no other files waiting for user confirmation to upload
                    const shouldAutoUpload = uppy.getFiles().length === 0

                    uppy.addFile({
                        name: 'pasted.txt',
                        type: 'text/plain',
                        data: new Blob([text], { type: 'text/plain' }),
                        source: 'Local',
                        isRemote: false,
                    })

                    if (shouldAutoUpload) {
                        uppy.upload()
                    }

                    dashboard.openModal()
                }
            }
        }

        // ensure a file has been pasted
        if (!cbData.files || cbData.files.length <= 0) {
            return
        }

        // pass event to the dashboard for handling
        dashboard.handlePaste(event)
        dashboard.openModal()
    }
}
