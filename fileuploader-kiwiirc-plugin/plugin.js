// polyfill globals for uppy on IE11
import 'core-js/features/array/iterator'
// import 'core-js/features/promise' // already included by kiwiirc

import Uppy from '@uppy/core'
import Dashboard from '@uppy/dashboard'
import Tus from '@uppy/tus'
import Webcam from '@uppy/webcam'
import '@uppy/core/dist/style.css'
import '@uppy/dashboard/dist/style.css'
import '@uppy/webcam/dist/style.css'
import Translator from '@uppy/utils/lib/Translator'
import sidebarFileList from './components/SidebarFileList.vue'
import isPromise from 'p-is-promise'
import TokenManager from './token-manager';

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

kiwi.plugin('fileuploader', function (kiwi, log) {
    // exposed api object
    kiwi.fileuploader = {}

    function setDefaultSetting(settingKey, value) {
        const fullKey = `settings.${settingKey}`
        if (kiwi.state.getSetting(fullKey) === undefined) {
            kiwi.state.setSetting(fullKey, value)
        }
    }

    // default settings
    setDefaultSetting('fileuploader.maxFileSize', 10 * MiB)
    setDefaultSetting('fileuploader.server', '/files')
    setDefaultSetting('fileuploader.textPastePromptMinimumLines', 5)
    setDefaultSetting('fileuploader.textPasteNeverPrompt', false)

    // add button to input bar
    const uploadFileButton = document.createElement('i')
    uploadFileButton.className = 'upload-file-button fa fa-upload'

    kiwi.addUi('input', uploadFileButton)

    let c = new kiwi.Vue(sidebarFileList)
    c.$mount()
    kiwi.addUi('about_buffer', c.$el, { title: 'Shared Files' })

    const tokenManager = new TokenManager()

    const uppy = Uppy({
        autoProceed: false,
        onBeforeFileAdded: (currentFile, files) => {
            // throws if invalid, canceling the file add
            getValidUploadTarget()
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
            maxFileSize: kiwi.state.setting('fileuploader.maxFileSize'),
        },
    })
        .use(Dashboard, {
            trigger: uploadFileButton,
            proudlyDisplayPoweredByUppy: false,
            closeModalOnClickOutside: true,
            note: kiwi.state.setting('fileuploader.note'),
        })
        .use(Webcam, { target: Dashboard })
        .use(Tus, {
            endpoint: kiwi.state.setting('fileuploader.server'),
            chunkSize: 512 * KiB,
        })
        .run()

    const dashboard = uppy.getPlugin('Dashboard')

    function loadLocale(_lang) {
        try {
            let lang = (_lang ? _lang : kiwi.i18n.language).split('-')
            let langUppy = lang[0] + '_' + lang[1].toUpperCase()
            import(
                /* webpackMode: "eager" */
                `@uppy/locales/lib/${langUppy}.js`
            ).then(locale => {
                setLocale(locale)
            });
        } catch (e) {
            import(
                /* webpackMode: "eager" */
                `@uppy/locales/lib/en_US.js`
            ).then(locale => {
                setLocale(locale)
            });
        }
    }

    function setLocale(locale) {
        // update uppy core
        uppy.opts.locale = locale
        uppy.translator = new Translator([ uppy.defaultLocale, uppy.opts.locale ])
        uppy.locale = uppy.translator.locale
        uppy.i18n = uppy.translator.translate.bind(uppy.translator)
        uppy.i18nArray = uppy.translator.translateArray.bind(uppy.translator)
        uppy.setState()

        // update uppy plugins
        uppy.iteratePlugins(function(plugin) {
            if (plugin.translator) {
                plugin.translator = uppy.translator
            }
            if (plugin.i18n) {
                plugin.i18n = uppy.i18n
            }
            if (plugin.i18nArray) {
                plugin.i18nArray = uppy.i18nArray
            }
            plugin.setPluginState()
        });
    }

    loadLocale()
    kiwi.state.$watch('user_settings.language', (lang) => {
        loadLocale(lang)
    });

    // expose plugin api
    kiwi.fileuploader.uppy = uppy
    kiwi.fileuploader.dashboard = dashboard

    // show uppy modal whenever a file is dragged over the page
    window.addEventListener('dragenter', event => {
        // swallow error and ignore drag if no valid buffer to share to
        try {
            getValidUploadTarget()
        } catch (err) {
            return
        }

        dashboard.openModal()
    })

    // show uppy modal when files are pasted
    kiwi.on('buffer.paste', event => {
        // swallow error and ignore paste if no valid buffer to share to
        try {
            getValidUploadTarget()
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
            const promptDisabled = kiwi.state.setting('fileuploader.textPasteNeverPrompt')
            if (promptDisabled) {
                return
            }
            const minLines = kiwi.state.setting('fileuploader.textPastePromptMinimumLines')
            const network = kiwi.state.getActiveNetwork()
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
    })

    uppy.on('file-added', file => {
        file.kiwiFileUploaderTargetBuffer = getValidUploadTarget()
    })

    uppy.on('upload-success', (file, response) => {
        // append filename to uploadURL
        const url = `${response.uploadURL}/${encodeURIComponent(file.meta.name)}`

        // emit a global kiwi event
        kiwi.emit('fileuploader.uploaded', { url, file })

        // send a message with the url of each successful upload
        const buffer = file.kiwiFileUploaderTargetBuffer
        buffer.say(`Uploaded file: ${url}`)
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
