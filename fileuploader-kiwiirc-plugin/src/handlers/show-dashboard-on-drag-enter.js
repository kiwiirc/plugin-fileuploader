import * as config from '@/config.js';

import { getValidUploadTarget } from '../utils/get-valid-upload-target';

export function showDashboardOnDragEnter(kiwiApi, dashboard) {
    return function handleDragEnter(/* event */) {
        // swallow error and ignore drag if no valid buffer to share to
        try {
            const buffer = getValidUploadTarget(kiwiApi);
            if (config.setting('userAccountsOnly') && !buffer.getNetwork().currentUser().account) {
                return;
            }
        } catch (err) {
            return;
        }

        dashboard.openModal();
    };
}
