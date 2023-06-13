import Uppy from '@uppy/core';
import Dashboard from '@uppy/dashboard';
import Tus from '@uppy/tus';
import Webcam from '@uppy/webcam';
import Audio from '@uppy/audio';
import ImageEditor from '@uppy/image-editor';
import prettierBytes from '@transloadit/prettier-bytes';

import Bytes from 'bytes';
import Wildcard from 'wildcard';

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

    const maxSizeTypesConfig = kiwiApi.state.setting('fileuploader.maxFileSizeTypes');
    const maxSizeTypes = Object.create(null);

    if (maxSizeTypesConfig) {
        Object.entries(maxSizeTypesConfig).forEach(([key, val]) => (maxSizeTypes[key] = Bytes.parse(val)));
    }

    const effectiveUppyOpts = {
        autoProceed: false,
        onBeforeFileAdded: (file) => {
            const buffer = kiwiApi.state.getActiveBuffer();
            const isValidTarget = buffer && (buffer.isChannel() || buffer.isQuery());
            if (!isValidTarget) {
                // TODO add translation
                uppy.info('Files can only be shared in channels or queries.', 'error', uppy.opts.infoTimeout);
                return false;
            }
            file.kiwiFileUploaderTargetBuffer = buffer;

            if (!file.type) {
                return true;
            }

            const matched = Object.entries(maxSizeTypes).find(([key]) => Wildcard(key, file.type));
            if (!matched) {
                return true;
            }

            if (file.size && file.size > matched[1]) {
                uppy.info(
                    uppy.i18n('exceedsSize', {
                        size: prettierBytes(matched[1]),
                        file: file.name,
                    }),
                    'error',
                    uppy.opts.infoTimeout
                );
                return false;
            }
            return true;
        },
        restrictions: {
            maxFileSize: Bytes.parse(kiwiApi.state.setting('fileuploader.maxFileSize')),
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
