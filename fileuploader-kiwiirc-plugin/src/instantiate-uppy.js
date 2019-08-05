import Uppy from '@uppy/core'
import Dashboard from '@uppy/dashboard'
import Tus from '@uppy/tus'
import Webcam from '@uppy/webcam'

import { KiB } from './constants/data-size'
import acquireExtjwtBeforeUpload from './handlers/uppy/acquire-extjwt-before-upload'
import { getValidUploadTarget } from './utils/get-valid-upload-target'

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
    }

    const effectiveTusOpts = {
        endpoint: kiwiApi.state.setting('fileuploader.server'),
        chunkSize: 512 * KiB,
        ...tusOptions,
    }

    const { handlerContext, handleBeforeUpload } = acquireExtjwtBeforeUpload(tokenManager)

    const effectiveUppyOpts = {
        autoProceed: false,
        onBeforeFileAdded: (/* currentFile, files */) => {
            // throws if invalid, cancelling the file add
            getValidUploadTarget(kiwiApi)
        },
        onBeforeUpload: handleBeforeUpload,
        restrictions: {
            maxFileSize: kiwiApi.state.setting('fileuploader.maxFileSize'),
        },
        ...uppyOptions,
    }

    const uppy = Uppy(effectiveUppyOpts)
        .use(Dashboard, effectiveDashboardOpts)
        .use(Webcam, { target: Dashboard })
        .use(Tus, effectiveTusOpts)
        .run()

    handlerContext.uppy = uppy // needs reference to uppy which didn't exist until now

    const dashboard = uppy.getPlugin('Dashboard')

    return { uppy, dashboard }
}
