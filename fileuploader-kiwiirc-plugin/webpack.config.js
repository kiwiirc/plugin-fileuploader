const path = require('path');
const CompressionPlugin = require('compression-webpack-plugin');
const { VueLoaderPlugin } = require('vue-loader');
const { CleanWebpackPlugin } = require('clean-webpack-plugin');

const ConvertLocalesPlugin = require('./build/convert-locales');

const makeSourceMap = process.argv.indexOf('--srcmap') > -1;
const shouldCompress = /\.(js|css|html|svg)(\.map)?$/;

module.exports = {
    mode: 'production',
    entry: './src/fileuploader-entry.js',
    output: {
        filename: 'plugin-fileuploader.js',
        path: path.resolve(__dirname, 'dist'),
    },
    resolve: {
        alias: {
            '@': path.resolve(__dirname, 'src'),
        },
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
            },
            {
                test: /\.css$/,
                use: [
                    { loader: 'style-loader' },
                    { loader: 'css-loader' },
                ],
            },
        ],
    },
    plugins: [
        new CleanWebpackPlugin(),
        new ConvertLocalesPlugin(),
        new VueLoaderPlugin(),
        new CompressionPlugin({
            test: shouldCompress,
        }),
    ],
    devtool: makeSourceMap ? 'source-map' : undefined,
    devServer: {
        static: {
            directory: path.join(__dirname, 'dist'),
            watch: false,
        },
        compress: true,
        host: process.env.HOST || 'localhost',
        port: process.env.PORT || 9001,
        headers: {
            // required for loading locales with XMLHttpRequest
            'Access-Control-Allow-Origin': '*',
        },
    },
    optimization: {
        minimize: true,
    },
    performance: {
        maxAssetSize: 1024000,
        maxEntrypointSize: 1024000,
        assetFilter: assetFilename =>
            !assetFilename.match(/\.map(\.(gz|br))?$/),
    },
};
