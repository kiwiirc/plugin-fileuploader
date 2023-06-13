/* global kiwi:true */
/* global _:true */

import Bytes from 'bytes';

const configBase = 'plugin-fileuploader';

const defaultConfig = {
    server: '/files/',
    maxFileSize: Bytes.parse('10MB'),
    uploadMessage: 'Uploaded file: %URL%',
    note: '',
    localePath: '',
    textPastePromptMinimumLines: 5,
    textPasteNeverPrompt: false,
    bufferInfoUploads: true,
    allowedFileTypes: null,
    maxFileSizeTypes: null,
};

export function setDefaults() {
    const oldConfig = kiwi.state.getSetting('settings.fileuploader');
    if (oldConfig) {
        console.warn('[DEPRECATION] Please update your fileuploader config to use "plugin-fileuploader" as its object key');
        kiwi.setConfigDefaults(configBase, oldConfig);
    }

    kiwi.setConfigDefaults(configBase, defaultConfig);
}

export function setting(name) {
    return kiwi.state.setting(_.compact([configBase, name]).join('.'));
}

export function getSetting(name) {
    return kiwi.state.getSetting(_.compact(['settings', configBase, name]).join('.'));
}

export function setSetting(name, value) {
    return kiwi.state.setSetting(_.compact(['settings', configBase, name]).join('.'), value);
}

export function getBasePath() {
    const scripts = document.getElementsByTagName('script');
    const scriptPath = scripts[scripts.length - 1].src;
    return scriptPath.substring(0, scriptPath.lastIndexOf('/') + 1);
}
