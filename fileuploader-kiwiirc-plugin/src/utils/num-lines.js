// doesn't count a final trailing newline as starting a new line
export function numLines(str) {
    const re = /\r?\n/g;
    let lines = 1;
    while (re.exec(str)) {
        if (re.lastIndex < str.length) {
            lines += 1;
        }
    }
    return lines;
}
