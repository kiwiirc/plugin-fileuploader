export default function acquireExtjwtBeforeUpload(tokenManager) {
    const handlerContext = {};

    function handleBeforeUpload(files) {
        const awaitingPromises = new Set();

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
            Promise.all(awaitingPromises.values()).then(() => {
                console.debug('Token acquisition complete. Restarting upload.');
                // Now all tokens are ready and waiting uppy.upload()
                // will cause this function to be executed again
                handlerContext.uppy.upload();
            });

            // Prevent uploads from starting until promises are resolved
            return false;
        }
    }

    return { handlerContext, handleBeforeUpload };
}
