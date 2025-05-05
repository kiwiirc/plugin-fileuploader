const fs = require('fs');

const utils = require('../../utils');

// class ConvertLocalesPlugin {
//     apply(compiler) {
//         compiler.hooks.emit.tap(this.constructor.name, async (compilation) => {
//             const compileOutput = compilation.options.output.path;
//             const outputDir = utils.pathResolve(compileOutput, 'plugin-fileuploader/locales/uppy/');
//             const sourceDir = utils.pathResolve('node_modules/@uppy/locales/lib');

//             fs.mkdirSync(outputDir, { recursive: true });

//             const files = fs.readdirSync(sourceDir).filter((f) => /\.js$/.test(f));
//             files.forEach((srcFile) => {
//                 const outFile = utils.pathResolve(outputDir, srcFile.toLowerCase() + 'on');
//                 console.log('out', outFile);
//                 import('file://' + utils.pathResolve(sourceDir, srcFile)).then((locale) => {
//                     fs.writeFileSync(outFile, JSON.stringify(locale.default.strings, null, 4));
//                 });
//             });
//         });
//     }
// }

class ConvertLocalesPlugin {
    apply(compiler) {
        const pluginName = this.constructor.name;
        const fileDependencies = new Set();

        compiler.hooks.beforeRun.tapAsync(pluginName, (compilation, callback) => {
            // run in build mode
            const compileOutput = compilation.options.output.path;
            const outputDir = utils.pathResolve(compileOutput, 'plugin-fileuploader/locales/uppy/');
            convertLocales(fileDependencies, outputDir, callback);
        });

        compiler.hooks.watchRun.tapAsync(pluginName, (compilation, callback) => {
            // run in dev mode
            const compileOutput = compilation.options.output.path;
            const outputDir = utils.pathResolve(compileOutput, 'plugin-fileuploader/locales/uppy/');
            convertLocales(fileDependencies, outputDir, callback);
        });

        compiler.hooks.afterEmit.tapAsync(pluginName, (compilation, callback) => {
            // Add file dependencies
            fileDependencies.forEach((dependency) => {
                compilation.fileDependencies.add(dependency);
            });

            callback();
        });
    }
}

async function convertLocales(fileDependencies, outputDir, callback) {
    fileDependencies.clear();
    const sourceDir = utils.pathResolve('node_modules/@uppy/locales/lib');
    const awaitPromises = new Set();

    fs.mkdirSync(outputDir, { recursive: true });

    const files = fs.readdirSync(sourceDir).filter((f) => /\.js$/.test(f));
    files.forEach((srcFile) => {
        const outFile = utils.pathResolve(outputDir, srcFile.toLowerCase() + 'on');
        const promise = import('file://' + utils.pathResolve(sourceDir, srcFile)).then((locale) => {
            if (!locale?.default?.strings) {
                return;
            }
            writeIfChanged(outFile, JSON.stringify(locale.default.strings, null, 4));
        });
        awaitPromises.add(promise);
    });

    await Promise.all(awaitPromises);
    callback();
}

function writeIfChanged(file, _data) {
    const data = Buffer.from(_data);
    if (fs.existsSync(file) && data.equals(fs.readFileSync(file))) {
        return;
    }

    fs.writeFileSync(file, data);
}

module.exports = ConvertLocalesPlugin;
