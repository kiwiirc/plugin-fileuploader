export function friendlyUrl(file, response = file.response) {
    // append filename to uploadURL
    return `${response.uploadURL}/${encodeURIComponent(file.meta.name)}`
}
