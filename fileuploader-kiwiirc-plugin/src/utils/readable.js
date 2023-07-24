/* global kiwi:true */

const TextFormatting = kiwi.require('helpers/TextFormatting');

export function bytesReadable(_bytes) {
    let bytes = _bytes;
    let idx = 0;
    const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB', 'EiB', 'ZiB', 'YiB'];
    while (bytes >= 1024 && idx < units.length) {
        bytes /= 1024;
        idx++;
    }
    return `${Math.round((bytes + Number.EPSILON) * 100) / 100} ${units[idx]}`;
}

export function durationReadable(timeSeconds) {
    // taken and modified from
    // https://github.com/kiwiirc/kiwiirc/blob/314420e2041d79ab7da8172e12ba1fd2d7709d32/src/helpers/TextFormatting.js#L198
    let seconds = timeSeconds;

    if (seconds <= 60) {
        return '< ' + TextFormatting.t('minute', { count: 1 });
    }

    const weeks = Math.floor(seconds / 604800);
    seconds -= weeks * 604800;

    const days = Math.floor(seconds / 86400);
    seconds -= days * 86400;

    const hours = Math.floor(seconds / 3600);
    seconds -= hours * 3600;

    const minutes = Math.floor(seconds / 60);
    seconds -= minutes * 60;

    const tmp = [];
    (weeks) && tmp.push(TextFormatting.t('week', { count: weeks }));
    (days) && tmp.push(TextFormatting.t('day', { count: days }));
    (hours) && tmp.push(TextFormatting.t('hour', { count: hours }));
    (minutes) && tmp.push(TextFormatting.t('minute', { count: minutes }));

    return tmp.join(', ');
}
