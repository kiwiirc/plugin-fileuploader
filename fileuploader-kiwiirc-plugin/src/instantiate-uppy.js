import Uppy from '@uppy/core';
import Dashboard from '@uppy/dashboard';
import Tus from '@uppy/tus';
import Webcam from '@uppy/webcam';
import Audio from '@uppy/audio';
import ImageEditor from '@uppy/image-editor';

import { KiB } from './constants/data-size';
import acquireExtjwtBeforeUpload from './handlers/uppy/acquire-extjwt-before-upload';

export default function instantiateUppy({
    kiwiApi,
    tokenManager,
    uploadFileButton,
    dashboardOptions,
    tusOptions,
    uppyOptions,
}) {
    const effectiveDashboardOpts = {
        trigger: uploadFileButton,
        proudlyDisplayPoweredByUppy: false,
        closeModalOnClickOutside: true,
        note: kiwiApi.state.setting('fileuploader.note'),
        ...dashboardOptions,
    };

    const effectiveTusOpts = {
        endpoint: kiwiApi.state.setting('fileuploader.server'),
        chunkSize: 512 * KiB,
        ...tusOptions,
    };

    const effectiveUppyOpts = {
        autoProceed: false,
        onBeforeFileAdded: (file) => {
            const buffer = kiwiApi.state.getActiveBuffer();
            const isValidTarget = buffer && (buffer.isChannel() || buffer.isQuery());
            if (!isValidTarget) {
                // TODO add translation
                uppy.info('Files can only be shared in channels or queries.', 'error', 5000);
                return false;
            }
            file.kiwiFileUploaderTargetBuffer = buffer;
            return true;
        },
        restrictions: {
            maxFileSize: kiwiApi.state.setting('fileuploader.maxFileSize'),
            allowedFileTypes: kiwiApi.state.setting('fileuploader.allowedFileTypes'),
        },
        ...uppyOptions,
    };

    const uppy = new Uppy(effectiveUppyOpts)
        .use(Dashboard, effectiveDashboardOpts)
        .use(Webcam, { target: Dashboard })
        .use(Audio, { target: Dashboard })
        .use(ImageEditor, {
            target: Dashboard,
            quality: 1,
        })
        .use(Tus, effectiveTusOpts);

    uppy.addPreProcessor(
        acquireExtjwtBeforeUpload(uppy, tokenManager)
    );

    const dashboard = uppy.getPlugin('Dashboard');

    return { uppy, dashboard };
}
