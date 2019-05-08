const path = require('path');
const VueLoaderPlugin = require('vue-loader/lib/plugin');
const CompressionPlugin = require('compression-webpack-plugin');
const BrotliPlugin = require('brotli-webpack-plugin');
const CleanWebpackPlugin = require('clean-webpack-plugin');

const shouldCompress = /\.(js|css|html|svg)(\.map)?$/

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
                    presets: [
                        ['@babel/preset-env', {
                            useBuiltIns: 'usage',
                            corejs: 3,
                        }],
                    ],
                },
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
        new CleanWebpackPlugin,
        new VueLoaderPlugin,
        new CompressionPlugin({
            test: shouldCompress,
        }),
        new BrotliPlugin({
            asset: '[path].br[query]',
            test: shouldCompress,
            threshold: 10240,
            minRatio: 0.8,
            deleteOriginalAssets: false,
        }),
    ],
    devtool: 'source-map',
    devServer: {
        filename: 'plugin-fileuploader.js',
        contentBase: path.join(__dirname, 'dist'),
        compress: true,
        host: process.env.HOST || 'localhost',
        port: process.env.PORT || 41040,
    },
    optimization: {
        minimize: true,
    },
    performance: {
        assetFilter: assetFilename =>
          !assetFilename.match(/\.map(\.(gz|br))?$/),
    },
}
