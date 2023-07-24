import { friendlyUrl } from '../../utils/friendly-url';
import { decodeMetadata } from '../../utils/decode-metadata';

export function shareCompletedUploadUrl(kiwiApi) {
    return function handleUploadSuccess(file, response) {
        const url = friendlyUrl(file, response);
        fetch(url, {
            method: 'HEAD',
        }).then((headResp) => {
            if (headResp.status !== 200 && headResp.status !== 412) {
                // old server instance responds with 412 Precondition Failed
                return;
            }
            sendUploadEvent(kiwiApi, url, file, headResp);
        }).catch(() => {
            sendUploadEvent(kiwiApi, url, file);
        });
    };
}

function sendUploadEvent(kiwiApi, url, file, headResp) {
    let rawMetadata;
    if (headResp) {
        rawMetadata = headResp.headers.get('upload-metadata');
    }

    let metadata = {};
    if (rawMetadata) {
        metadata = decodeMetadata(rawMetadata);
    }

    // emit a global kiwi event
    kiwiApi.emit('fileuploader.uploaded', { url, file, metadata });

    const buffer = file.kiwiFileUploaderTargetBuffer;
    const tagData = {
        size: file.size,
        type: file.type,
    };
    if (metadata.expires) {
        tagData.expires = parseInt(metadata.expires, 10);
    }

    const msgTemplate = kiwiApi.state.getSetting('settings.fileuploader.uploadMessage');
    const message = msgTemplate.replace('%URL%', url);

    buffer.say(message, {
        tags: { '+kiwiirc.com/fileuploader': JSON.stringify(tagData) },
    });
}
