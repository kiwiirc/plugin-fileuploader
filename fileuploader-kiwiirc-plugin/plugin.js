import { Core as Uppy, Dashboard, Tus, Webcam } from 'uppy'
import 'uppy/dist/uppy.min.css'
import fileListComponent from './components/fileListComponent.vue'

const KiB = 2 ** 10
const MiB = 2 ** 20

function getValidUploadTarget() {
    const buffer = kiwi.state.getActiveBuffer()
    const isValidTarget = buffer && (buffer.isChannel() || buffer.isQuery())
    if (!isValidTarget) {
        throw new Error('Files can only be shared in channels or queries.')
    }
    return buffer
}

// doesn't count a final trailing newline as starting a new line
function numLines(str) {
    const re = /\r?\n/g
    let lines = 1
    while (re.exec(str)) {
        if (re.lastIndex < str.length) {
            lines += 1
        }
    }
    return lines
}

kiwi.plugin('fileuploader', function(kiwi, log) {
    // exposed api object
    kiwi.fileuploader = {}

    const settings = kiwi.state.setting('fileuploader')

    // add button to input bar
    const uploadFileButton = document.createElement('i')
    uploadFileButton.className = 'upload-file-button fa fa-upload'

    kiwi.addUi('input', uploadFileButton)

    // add button to input bar
    const historyButton = document.createElement('i')
    historyButton.className = 'history-button fa fa-history'

    kiwi.addUi('input', historyButton)

    historyButton.onclick = e => {
        kiwi.emit("sidebar.show")
        kiwi.showInSidebar(fileListComponent)
    }

    let fileList = []

    kiwi.on('message.new', e => {
        if (e.message.indexOf(settings.server) !== -1) {
            let currentdate = new Date(); 
            let time = currentdate.getHours() + ":"  
                + currentdate.getMinutes() + ":" 
                + currentdate.getSeconds();
            let link = {
                url: e.message.substring(e.message.indexOf(settings.server)),
                nick: e.nick,
                time 
            };
            link.url = link.url.split(' ')[0].split(')')[0]
            fileList.push(link);
        }
    })

    const uppy = Uppy({
        autoProceed: false,
        onBeforeFileAdded: (currentFile, files) => {
            // throws if invalid, canceling the file add
            getValidUploadTarget()
        },
        restrictions: {
            maxFileSize: settings.maxFileSize || 10 * MiB,
        },
    })
        .use(Dashboard, {
            trigger: uploadFileButton,
            proudlyDisplayPoweredByUppy: false,
            closeModalOnClickOutside: true,
            note: settings.note,
        })
        .use(Webcam, { target: Dashboard })
        .use(Tus, {
            endpoint: settings.server || '/files',
            chunkSize: 512 * KiB,
        })
        .run()

    const dashboard = uppy.getPlugin('Dashboard')

    // expose plugin api
    kiwi.fileuploader.uppy = uppy
    kiwi.fileuploader.dashboard = dashboard

    // show uppy modal whenever a file is dragged over the page
    window.addEventListener('dragenter', event => {
        // swallow error and ignore drag if no valid buffer to share to
        try { getValidUploadTarget() } catch (err) { return }

        dashboard.openModal()
    })

    // show uppy modal when files are pasted
    kiwi.on('buffer.paste', event => {
        // swallow error and ignore paste if no valid buffer to share to
        try { getValidUploadTarget() } catch (err) { return }

        // IE 11 puts the clipboardData on the window
        const cbData = event.clipboardData || window.clipboardData

        if (!cbData) {
            return
        }

        // detect large text pastes, offer to create a file upload instead
        const text = cbData.getData('text')
        if (text) {
            const network = kiwi.state.getActiveNetwork()
            const networkMaxLineLen = network.ircClient.options.message_max_length
            if (text.length > networkMaxLineLen || numLines(text) > 4) {
                const msg = 'You pasted a lot of text.\nWould you like to upload as a file instead?'
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
    })

    uppy.on('file-added', file => {
        file.kiwiFileUploaderTargetBuffer = getValidUploadTarget()
    })

    uppy.on('upload-success', (file, resp, uploadURL) => {
        // append filename to uploadURL
        uploadURL = `${uploadURL}/${encodeURIComponent(file.meta.name)}`
        uppy.setFileState(file.id, { uploadURL })
        file = uppy.getFile(file.id)

        // emit a global kiwi event
        kiwi.emit('fileuploader.uploaded', { url: uploadURL, file })

        // send a message with the url of each successful upload
        const buffer = file.kiwiFileUploaderTargetBuffer
        buffer.say(`Uploaded file: ${uploadURL}`)
    })

    uppy.on('complete', result => {
        // automatically close upload modal if all uploads succeeded
        if (result.failed.length === 0) {
            uppy.reset()
            // TODO: this would be nicer with a css transition: delay, then fade out
            dashboard.closeModal()
        }
    })
})
