export function getValidUploadTarget(kiwiApi) {
    const buffer = kiwiApi.state.getActiveBuffer();
    const isValidTarget = buffer && (buffer.isChannel() || buffer.isQuery());
    if (!isValidTarget) {
        throw new Error('Files can only be shared in channels or queries.');
    }
    return buffer;
}
