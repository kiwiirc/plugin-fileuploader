import '@uppy/core/dist/style.css';
import '@uppy/dashboard/dist/style.css';
import '@uppy/webcam/dist/style.css';
import '@uppy/audio/dist/style.css';
import '@uppy/image-editor/dist/style.css';
import sidebarFileList from './components/SidebarFileList.vue';
import audioPlayerComponent from './components/AudioPlayer.vue';
import { showDashboardOnDragEnter } from './handlers/show-dashboard-on-drag-enter';
import { uploadOnPaste } from './handlers/upload-on-paste';
import { closeModalWhenUploadsCompleted } from './handlers/uppy/close-modal-when-uploads-completed';
import { shareCompletedUploadUrl } from './handlers/uppy/share-completed-upload-url';
// import { trackFileUploadTarget } from './handlers/uppy/track-file-upload-target';
import instantiateUppy from './instantiate-uppy';
import instantiateUppyLocales from './instantiate-uppy-locales';
import { createPromptUpload } from './prompt-upload';
import TokenManager from './token-manager';

import * as config from '@/config.js';

/* global kiwi:true */
kiwi.plugin('fileuploader', function(kiwiApi, log) {
    // default settings
    config.setDefaults();

    // add button to input bar
    const uploadFileButton = document.createElement('i');
    uploadFileButton.className = 'upload-file-button fa fa-upload';
    kiwiApi.addUi('input', uploadFileButton);

    // add sidebar panel
    if (config.getSetting('bufferInfoUploads')) {
        kiwiApi.addUi('about_buffer', sidebarFileList, { title: 'Shared Files' });
    }

    // set up main uppy object
    const tokenManager = new TokenManager(kiwiApi);
    const { uppy, dashboard } = instantiateUppy({
        kiwiApi,
        tokenManager,
        uploadFileButton,
        dashboardOptions: config.getSetting('dashboardOptions'),
        tusOptions: config.getSetting('tusOptions'),
        uppyOptions: config.getSetting('uppyOptions'),
    });

    instantiateUppyLocales(kiwiApi, uppy);

    const promptUpload = createPromptUpload({ kiwiApi, tokenManager });
    // expose plugin api
    kiwiApi.fileuploader = { uppy, dashboard, promptUpload };

    // show uppy modal whenever a file is dragged over the page
    window.addEventListener('dragenter', showDashboardOnDragEnter(kiwiApi, dashboard));

    // show uppy modal when files are pasted
    kiwiApi.on('buffer.paste', uploadOnPaste(kiwiApi, uppy, dashboard));

    kiwiApi.on('message.new', (event) => {
        const message = event.message;
        if (!message.tags || !message.tags['+kiwiirc.com/fileuploader']) {
            return;
        }

        try {
            const fileInfo = kiwiApi.JSON5.parse(message.tags['+kiwiirc.com/fileuploader']);
            message.fileuploader = {
                hasError: false,
                isAudio: false,
                fileInfo,
            };

            if (fileInfo.type?.startsWith('audio/')) {
                message.bodyTemplate = audioPlayerComponent;
                message.fileuploader.isAudio = true;
            }
        } catch (err) {
            log.error('Failed to parse fileuploader message-tag', err);
        }
    });

    kiwiApi.on('message.prestyle', (event) => {
        const message = event.message;
        if (!message.fileuploader?.isAudio) {
            return;
        }
        message.embed.payload = '';
    });

    // send message with link to buffer when upload finishes
    uppy.on('upload-success', shareCompletedUploadUrl(kiwiApi));

    // hide dashboard after last upload finishes
    uppy.on('complete', closeModalWhenUploadsCompleted(uppy, dashboard));
});
