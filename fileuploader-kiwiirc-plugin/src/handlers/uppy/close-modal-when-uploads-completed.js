export function closeModalWhenUploadsCompleted(uppy, dashboard) {
    return function handleCompleted(result) {
        // automatically close upload modal if all uploads succeeded
        if (result.failed.length === 0) {
            uppy.reset()
            // TODO: this would be nicer with a css transition: delay, then fade out
            dashboard.closeModal()
        }
    }
}
