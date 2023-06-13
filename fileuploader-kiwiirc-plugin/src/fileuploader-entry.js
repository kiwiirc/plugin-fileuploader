// polyfill globals for uppy on IE11
import 'core-js/features/array/iterator';
// import 'core-js/features/promise' // already included by kiwiirc

import '@uppy/core/dist/style.css';
import '@uppy/dashboard/dist/style.css';
import '@uppy/webcam/dist/style.css';
import '@uppy/audio/dist/style.css';
import '@uppy/image-editor/dist/style.css';

import sidebarFileList from './components/SidebarFileList.vue';
import audioPlayerComponent from './components/AudioPlayer.vue';
import { MiB } from './constants/data-size';
import { showDashboardOnDragEnter } from './handlers/show-dashboard-on-drag-enter';
import { uploadOnPaste } from './handlers/upload-on-paste';
import { closeModalWhenUploadsCompleted } from './handlers/uppy/close-modal-when-uploads-completed';
import { shareCompletedUploadUrl } from './handlers/uppy/share-completed-upload-url';
// import { trackFileUploadTarget } from './handlers/uppy/track-file-upload-target';
import instantiateUppy from './instantiate-uppy';
import instantiateUppyLocales from './instantiate-uppy-locales';
import { createPromptUpload } from './prompt-upload';
import TokenManager from './token-manager';
import { setDefaultSetting } from './utils/set-default-setting';

let scriptPath;

(function() {
    const scriptElements = document.getElementsByTagName('script');
    const thisScriptSrc = scriptElements[scriptElements.length - 1].src;
    scriptPath = thisScriptSrc.substring(0, thisScriptSrc.lastIndexOf('/') + 1);
})();

/* global kiwi:true */
kiwi.plugin('fileuploader', function(kiwiApi, log) {
    // default settings
    setDefaultSetting(kiwiApi, 'fileuploader.allowedFileTypes', null);
    setDefaultSetting(kiwiApi, 'fileuploader.maxFileSize', 10 * MiB);
    setDefaultSetting(kiwiApi, 'fileuploader.server', '/files/');
    setDefaultSetting(kiwiApi, 'fileuploader.textPastePromptMinimumLines', 5);
    setDefaultSetting(kiwiApi, 'fileuploader.textPasteNeverPrompt', false);
    setDefaultSetting(kiwiApi, 'fileuploader.bufferInfoUploads', true);
    setDefaultSetting(kiwiApi, 'fileuploader.localePath', '');
    setDefaultSetting(kiwiApi, 'fileuploader.uploadMessage', 'Uploaded file: %URL%');

    // add button to input bar
    const uploadFileButton = document.createElement('i');
    uploadFileButton.className = 'upload-file-button fa fa-upload';
    kiwiApi.addUi('input', uploadFileButton);

    // add sidebar panel
    if (kiwiApi.state.setting('fileuploader.bufferInfoUploads')) {
        kiwiApi.addUi('about_buffer', sidebarFileList, { title: 'Shared Files' });
    }

    // set up main uppy object
    const tokenManager = new TokenManager(kiwiApi);
    const { uppy, dashboard } = instantiateUppy({
        kiwiApi,
        tokenManager,
        uploadFileButton,
    });

    instantiateUppyLocales(kiwiApi, uppy, scriptPath);

    const promptUpload = createPromptUpload({ kiwiApi, tokenManager });
    // expose plugin api
    kiwiApi.fileuploader = { uppy, dashboard, promptUpload };

    // show uppy modal whenever a file is dragged over the page
    window.addEventListener('dragenter', showDashboardOnDragEnter(kiwiApi, dashboard));

    // show uppy modal when files are pasted
    kiwiApi.on('buffer.paste', uploadOnPaste(kiwiApi, uppy, dashboard));

    kiwiApi.on('message.new', (event, network, eventObj) => {
        if (!event.message.tags || !event.message.tags['+kiwiirc.com/fileuploader']) {
            return;
        }

        const fileInfo = kiwiApi.JSON5.parse(event.message.tags['+kiwiirc.com/fileuploader']);
        if (!fileInfo.type || !fileInfo.type.startsWith('audio/')) {
            return;
        }

        const message = event.message;
        message.bodyTemplate = audioPlayerComponent;
        console.log('message.new', event);
    });

    // send message with link to buffer when upload finishes
    uppy.on('upload-success', shareCompletedUploadUrl(kiwiApi));

    // hide dashboard after last upload finishes
    uppy.on('complete', closeModalWhenUploadsCompleted(uppy, dashboard));
});
