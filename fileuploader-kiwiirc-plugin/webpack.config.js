const path = require('path')
const VueLoaderPlugin = require('vue-loader/lib/plugin')
const CompressionPlugin = require('compression-webpack-plugin')
const { CleanWebpackPlugin } = require('clean-webpack-plugin')

const makeSourceMap = process.argv.indexOf('--srcmap') > -1;
const shouldCompress = /\.(js|css|html|svg)(\.map)?$/

module.exports = {
    mode: 'production',
    entry: './src/fileuploader-entry.js',
    output: {
        filename: 'plugin-fileuploader.js',
        path: path.resolve(__dirname, "dist"),
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
        new CleanWebpackPlugin,
        new VueLoaderPlugin,
        new CompressionPlugin({
            test: shouldCompress,
        }),
    ],
    devtool: makeSourceMap ? 'source-map' : undefined,
    devServer: {
        compress: true,
        host: process.env.HOST || 'localhost',
        port: process.env.PORT || 41040,
    },
    optimization: {
        minimize: true,
    },
    performance: {
        maxAssetSize: 700000,
        maxEntrypointSize: 700000,
        assetFilter: assetFilename =>
          !assetFilename.match(/\.map(\.(gz|br))?$/),
    },
}
