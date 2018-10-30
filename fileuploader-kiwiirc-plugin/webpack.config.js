const path = require('path');
const VueLoaderPlugin = require('vue-loader/lib/plugin');

module.exports = {
    mode: 'production',
    entry: './plugin.js',
    output: {
        filename: 'plugin-fileuploader.js',
    },
    module: {
        rules: [
            {
                test: /\.vue$/,
                loader: 'vue-loader',
            },
            {
                test: /\.js$/,
                exclude: /node_modules/,
                loader: 'babel-loader',
                query: {
                    presets: ['env'],
                }
            },
            {
                test: /\.css$/,
                use: [ 'style-loader', 'css-loader' ]
            }
        ]
    },
    plugins: [
        new VueLoaderPlugin
    ],
    devtool: 'source-map',
    devServer: {
        filename: 'plugin-fileuploader.js',
        contentBase: path.join(__dirname, 'dist'),
        compress: true,
        host: process.env.HOST || 'localhost',
        port: 41040,
    }
};
