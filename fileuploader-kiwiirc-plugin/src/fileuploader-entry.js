// polyfill globals for uppy on IE11
import 'core-js/features/array/iterator'
// import 'core-js/features/promise' // already included by kiwiirc

import '@uppy/core/dist/style.css'
import '@uppy/dashboard/dist/style.css'
import '@uppy/webcam/dist/style.css'

import sidebarFileList from './components/SidebarFileList.vue'
import { MiB } from './constants/data-size'
import { showDashboardOnDragEnter } from './handlers/show-dashboard-on-drag-enter';
import { uploadOnPaste } from './handlers/upload-on-paste'
import { closeModalWhenUploadsCompleted } from './handlers/uppy/close-modal-when-uploads-completed';
import { shareCompletedUploadUrl } from './handlers/uppy/share-completed-upload-url';
import { trackFileUploadTarget } from './handlers/uppy/track-file-upload-target';
import instantiateUppy from './instantiate-uppy'
import { createPromptUpload } from './prompt-upload'
import TokenManager from './token-manager'
import { setDefaultSetting } from './utils/set-default-setting'

kiwi.plugin('fileuploader', function (kiwiApi, log) {
    // default settings
    setDefaultSetting(kiwiApi, 'fileuploader.allowedFileTypes', null)
    setDefaultSetting(kiwiApi, 'fileuploader.maxFileSize', 10 * MiB)
    setDefaultSetting(kiwiApi, 'fileuploader.server', '/files')
    setDefaultSetting(kiwiApi, 'fileuploader.textPastePromptMinimumLines', 5)
    setDefaultSetting(kiwiApi, 'fileuploader.textPasteNeverPrompt', false)

    // add button to input bar
    const uploadFileButton = document.createElement('i')
    uploadFileButton.className = 'upload-file-button fa fa-upload'
    kiwiApi.addUi('input', uploadFileButton)

    // add sidebar panel
    let c = new kiwiApi.Vue(sidebarFileList)
    c.$mount()
    kiwiApi.addUi('about_buffer', c.$el, { title: 'Shared Files' })

    // set up main uppy object
    const tokenManager = new TokenManager()
    const { uppy, dashboard } = instantiateUppy({
        kiwiApi,
        tokenManager,
        uploadFileButton,
    })

    const promptUpload = createPromptUpload({ kiwiApi, tokenManager })
    // expose plugin api
    kiwiApi.fileuploader = { uppy, dashboard, promptUpload }

    // show uppy modal whenever a file is dragged over the page
    window.addEventListener('dragenter', showDashboardOnDragEnter(kiwiApi, dashboard))

    // show uppy modal when files are pasted
    kiwiApi.on('buffer.paste', uploadOnPaste(kiwiApi, uppy, dashboard))

    // remember what buffer the upload is targetting
    uppy.on('file-added', trackFileUploadTarget(kiwiApi))

    // send message with link to buffer when upload finishes
    uppy.on('upload-success', shareCompletedUploadUrl(kiwiApi))

    // hide dashboard after last upload finishes
    uppy.on('complete', closeModalWhenUploadsCompleted(uppy, dashboard))
})
