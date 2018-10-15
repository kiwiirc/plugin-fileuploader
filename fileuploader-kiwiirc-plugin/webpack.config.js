const path = require('path');

module.exports = {
    mode: 'production',
    entry: './plugin.js',
    output: {
        filename: 'plugin-fileuploader.js',
    },
    module: {
        rules: [{
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
devtool: 'source-map',
devServer: {
    filename: 'plugin-fileuploader.js',
    contentBase: path.join(__dirname, 'dist'),
    compress: true,
    host: process.env.HOST || 'localhost',
    port: 41040,
}
};
