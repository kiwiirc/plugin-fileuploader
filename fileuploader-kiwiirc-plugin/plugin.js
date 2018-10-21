import { Core as Uppy, Dashboard, Tus, Webcam } from 'uppy'
import 'uppy/dist/uppy.min.css'

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

    let fileList = []

    var fileListComponent = kiwi.Vue.extend({
        template: `
            <div style="overflow: auto; background: #eee; height: 100%;">
                <div v-if="!fileList.length" style="margin: 10px; font-family: arial, tahoma;">
                    No new files have been uploaded!<br><br><br>Check back here to see the file history...
                </div>
                <div v-else style="margin: 10px; font-family: arial, tahoma;">
                    Recently uploaded files...<br><br>
                    <table style="width: 100%; border-collapse: collapse;">
                        <tr><th></th><th>Nick</th><th>Link</th><th>Time</th></tr>
                        <tr v-for="(file, idx) in fileList" :key="file" style="border-bottom: 1px solid #aaa;">
                            <td style="border-bottom: 1px solid #aaa;">
                                <button
                                    :href="file.url"
                                    @click="fileList.splice(idx,1)"
                                    style="background: #f88; color: #822; border: none; border-radius: 5px; padding-left: 5px; padding-right: 5px; font-size: 18px; cursor: pointer;"
                                >
                                    <i class="fa fa-trash" aria-hidden="true"/>
                                </button>
                            </td>
                            <td style="text-align: center; border-bottom: 1px solid #aaa;">
                                {{file.nick}}
                            </td>
                            <td style="border-bottom: 1px solid #aaa;">
                                <a
                                    :href="file.url"
                                    @click.prevent.stop="loadContent(file.url)"
                                    style="display: inline-block; text-align: center; width: 100%; border: none; background: #8f8; border-radius: 5px; text-decoration: underline; cursor: pointer;"
                                >
                                    {{getFileName(file.url)}}
                                </a>
                            </td>
                            <td style="text-align: center; border-bottom: 1px solid #aaa;">
                                {{file.time}}
                            </td>
                        </tr>
                    </table>
                </div>
            </div>
        `,
        data() {
            return {
                sidebarOpen: true,
                fileList,
            };
        },
        methods: {
            getFileName(file) {
                let name = file.split('/')[file.split('/').length-1];
                if (name.length >= 25) {
                    name = name.substring(0, 18) + '...' + name.substring(name.length - 4);
                }
                return name;
            },
            loadContent(url) {
                kiwi.emit('mediaviewer.show', url);
            },
        },
    });

    historyButton.onclick = e => {
        kiwi.emit("sidebar.show")
        kiwi.emit('sidebar.component', fileListComponent)
    }

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
