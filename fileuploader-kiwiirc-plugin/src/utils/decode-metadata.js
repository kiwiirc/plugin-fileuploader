export function decodeMetadata(header) {
    const metadata = Object.create(null);
    const elements = header.split(',');

    elements.forEach((element) => {
        const parts = element.trim().split(' ').filter((p) => !!p);

        if (!parts.length || parts.length > 2) {
            return;
        }

        const key = parts[0];
        if (!key) {
            return;
        }

        let value = '';
        if (parts.length === 2) {
            try {
                value = atob(parts[1]);
            } catch {
                return;
            }
        }

        metadata[key] = value;
    });

    return metadata;
}
