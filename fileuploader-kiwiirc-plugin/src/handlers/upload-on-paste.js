import { getValidUploadTarget } from '@/utils/get-valid-upload-target';
import { numLines } from '@/utils/num-lines';

// import PasteConfirm from '@/components/PasteConfirm.vue';

import * as config from '@/config.js';

export function uploadOnPaste(kiwiApi, uppy, dashboard) {
    return async function handleBufferPaste(event) {
        // swallow error and ignore paste if no valid buffer to share to
        try {
            const buffer = getValidUploadTarget(kiwiApi);
            if (config.setting('userAccountsOnly') && !buffer.getNetwork().currentUser().account) {
                return;
            }
        } catch (err) {
            return;
        }

        // IE 11 puts the clipboardData on the window
        const cbData = event.clipboardData || window.clipboardData;

        if (!cbData) {
            return;
        }

        // detect large text pastes, offer to create a file upload instead
        const text = cbData.getData('text');
        if (text) {
            const promptDisabled = config.getSetting('textPasteNeverPrompt');
            if (promptDisabled) {
                return;
            }
            const minLines = config.getSetting('textPastePromptMinimumLines');
            const network = kiwiApi.state.getActiveNetwork();
            const networkMaxLineLen =
                network.ircClient.options.message_max_length;
            if (text.length > networkMaxLineLen || numLines(text) >= minLines) {
                // console.log('too may lines', event);

                // event.preventDefault();
                // event.stopPropagation();

                // kiwiApi.state.$emit('input.tool', PasteConfirm);

                // const msg =
                //     'You pasted a lot of text.\nWould you like to upload as a file instead?';
                // eslint-disable-next-line no-alert
                // if (window.confirm(msg)) {
                //     // stop IrcInput from ingesting the pasted text
                //     event.preventDefault();
                //     event.stopPropagation();

                //     // only if there are no other files waiting for user confirmation to upload
                //     const shouldAutoUpload = uppy.getFiles().length === 0;

                //     uppy.addFile({
                //         name: 'pasted.txt',
                //         type: 'text/plain',
                //         data: new Blob([text], { type: 'text/plain' }),
                //         source: 'Local',
                //         isRemote: false,
                //     });

                //     if (shouldAutoUpload) {
                //         uppy.upload();
                //     } else {
                //         dashboard.openModal();
                //     }
                // }
            }

            return;
        }

        // ensure a file has been pasted
        if (!cbData.files || cbData.files.length <= 0) {
            return;
        }

        // pass event to the dashboard for handling
        dashboard.handlePaste(event);
        dashboard.openModal();
    };
}
