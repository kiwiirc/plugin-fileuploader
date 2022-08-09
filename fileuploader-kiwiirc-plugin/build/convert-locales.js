const fs = require('fs');
const path = require('path');

const pluginName = 'ConvertLocalesPlugin';

class ConvertLocalesPlugin {
    apply(compiler) {
        compiler.hooks.emit.tap(pluginName, (compilation) => {
            const outputDir = path.resolve(__dirname, compilation.options.output.path, 'plugin-fileuploader/locales/uppy/');
            const nodeDir = path.resolve(__dirname, '../node_modules');
            const sourceDir = path.resolve(__dirname, nodeDir, '@uppy/locales/lib');

            fs.mkdirSync(outputDir, { recursive: true });

            const files = fs.readdirSync(sourceDir).filter(f => /\.js$/.test(f));
            for (let i = 0; i < files.length; i++) {
                const srcFile = files[i];
                const outFile = path.resolve(__dirname, outputDir, srcFile.toLowerCase() + 'on');
                const locale = require(path.resolve(__dirname, sourceDir, srcFile));
                fs.writeFileSync(outFile, JSON.stringify(locale.strings, null, 4));
            }
        });
    }
}

module.exports = ConvertLocalesPlugin;
