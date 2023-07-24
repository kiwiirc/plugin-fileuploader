export function decodeMetadata(header) {
    const metadata = Object.create(null);
    const elements = header.split(',');

    for (const element of elements) {
        const parts = element.trim().split(' ').filter((p) => !!p);

        if (!parts.length || parts.length > 2) {
            continue;
        }

        const key = parts[0];
        if (!key) {
            continue;
        }

        let value = '';
        if (parts.length === 2) {
            try {
                value = atob(parts[1]);
            } catch {
                continue;
            }
        }

        metadata[key] = value;
    }

    return metadata;
}
