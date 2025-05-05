module.exports = {
    terserOptions: {
        compress: {
            booleans: true,
            conditionals: true,
            dead_code: true,
            evaluate: true,
            if_return: true,
            sequences: true,
            unused: true,
        },
    },
    extractComments: {
        condition: false,
    },
};
