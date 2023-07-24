export default function acquireExtjwtBeforeUpload(uppy, tokenManager) {
    function handleBeforeUpload(fileIDs) {
        const awaitingPromises = new Set();

        const files = fileIDs.map((id) => uppy.getFile(id));

        for (const fileObj of Object.values(files)) {
            const network = fileObj.kiwiFileUploaderTargetBuffer.getNetwork();
            if (network.ircClient.network.options.EXTJWT !== '1') {
                // The network does not support EXTJWT Version 1
                continue;
            }

            const maybeToken = tokenManager.get(network);
            if (maybeToken instanceof Promise) {
                awaitingPromises.add(maybeToken);
            } else {
                fileObj.meta.extjwt = maybeToken;
            }
        }

        if (awaitingPromises.size) {
            // Wait for the unresolved promises then resume uploading
            return Promise.all(awaitingPromises.values()).then(() => {
                console.debug('Token acquisition complete. Restarting upload.');
            });
        }
    }

    return handleBeforeUpload;
}
