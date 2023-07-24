import { getValidUploadTarget } from '../utils/get-valid-upload-target';

export function showDashboardOnDragEnter(kiwiApi, dashboard) {
    return function handleDragEnter(/* event */) {
        // swallow error and ignore drag if no valid buffer to share to
        try {
            getValidUploadTarget(kiwiApi);
        } catch (err) {
            return;
        }

        dashboard.openModal();
    };
}
