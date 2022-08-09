import { getValidUploadTarget } from '../../utils/get-valid-upload-target';

export function trackFileUploadTarget(kiwiApi) {
    return function handleFileAdded(file) {
        file.kiwiFileUploaderTargetBuffer = getValidUploadTarget(kiwiApi);
    };
}
