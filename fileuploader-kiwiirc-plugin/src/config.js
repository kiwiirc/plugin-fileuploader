/* global kiwi:true */

import Bytes from 'bytes';

const configBase = 'plugin-fileuploader';

const defaultConfig = {
    server: '/files/',
    maxFileSize: Bytes.parse('10MB'),
    uploadMessage: 'Uploaded file: %URL%',
    note: '',
    localePath: getBasePath() + 'plugin-fileuploader/locales/uppy/%LANG%.json',
    textPastePromptMinimumLines: 5,
    textPasteNeverPrompt: false,
    userAccountsOnly: false,
    bufferInfoUploads: true,
    allowedFileTypes: null,
    maxFileSizeTypes: null,
    dashboardOptions: {},
    tusOptions: {},
    uppyOptions: {},
};

export function setDefaults() {
    const oldConfig = kiwi.state.getSetting('settings.fileuploader');
    if (oldConfig) {
        // eslint-disable-next-line no-console, vue/max-len
        console.warn('[DEPRECATION] Please update your fileuploader config to use "plugin-fileuploader" as its object key');
        kiwi.setConfigDefaults(configBase, oldConfig);
    }

    kiwi.setConfigDefaults(configBase, defaultConfig);
}

export function setting(name, value) {
    return kiwi.state.setting([configBase, name].join('.'), value);
}

export function getSetting(name) {
    return kiwi.state.getSetting(['settings', configBase, name].join('.'));
}

export function setSetting(name, value) {
    return kiwi.state.setSetting(['settings', configBase, name].join('.'), value);
}

function getBasePath() {
    const scripts = document.getElementsByTagName('script');
    const scriptPath = scripts[scripts.length - 1].src;
    return scriptPath.substr(0, scriptPath.lastIndexOf('/') + 1);
}
