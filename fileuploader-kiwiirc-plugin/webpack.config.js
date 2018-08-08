const path = require('path');

module.exports = {
  mode: 'development',
  entry: './plugin.js',
  module: {
    rules: [{
      test: /\.js$/,
      exclude: /node_modules/,
      loader: 'babel-loader',
      query: {
        presets: ['es2015'],
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
      contentBase: path.join(__dirname, "dist"),
      compress: true,
      port: 9000
  }
};
