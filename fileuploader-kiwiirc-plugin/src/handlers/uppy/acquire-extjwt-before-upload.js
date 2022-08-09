import isPromise from 'p-is-promise';

export default function acquireExtjwtBeforeUpload(tokenManager) {
    const handlerContext = {};

    function handleBeforeUpload(files) {
        const uniqNetworks = new Set(
            Object.values(files)
                .map(file =>
                    file.kiwiFileUploaderTargetBuffer.getNetwork()
                )
        );
        console.log('files', files);
        console.log('uniqNetworks', uniqNetworks);
        const tokens = new Map();
        for (const network of uniqNetworks) {
            tokens.set(network, tokenManager.get(network));
        }

        const tokenPromises = [...tokens.values()].filter(isPromise);
        if (tokenPromises.length > 0) {
            console.debug(
                'Tokens were not available synchronously. ' +
                'Cancelling upload start and acquiring tokens.'
            );

            // restart uploads once all needed tokens are acquired
            Promise.all(tokenPromises).then(() => {
                console.debug('Token acquisition complete. Restarting upload.');
                handlerContext.uppy.upload();
            }).catch(err => {
                handlerContext.uppy.info({
                    message: 'Unhandled error acquiring EXTJWT tokens!',
                    details: err,
                }, 'error', 5000);
            });

            // prevent uploads from starting now
            return false;
        }

        // WARNING: don't use an immutable update pattern here!
        // if we return new objects, uppy will ignore our changes
        for (const fileObj of Object.values(files)) {
            const token = tokenManager.get(fileObj.kiwiFileUploaderTargetBuffer.getNetwork());
            if (token) { // token will be false if the server response was 'Unknown Command'
                fileObj.meta.extjwt = token;
            }
        }
    }

    return { handlerContext, handleBeforeUpload };
}
