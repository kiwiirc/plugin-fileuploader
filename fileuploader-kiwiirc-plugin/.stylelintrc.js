module.exports = {
    extends: ['stylelint-config-standard', 'stylelint-config-standard-scss', 'stylelint-config-recess-order'],
    overrides: [
        {
            files: ['**/*.vue', '**/*.html'],
            customSyntax: 'postcss-html',
        },
    ],
    rules: {
        'alpha-value-notation': null,
        'color-function-notation': null,
        'declaration-block-no-redundant-longhand-properties': null,
        'declaration-no-important': true,
        'media-feature-range-notation': null,
        'no-descending-specificity': null,
        'no-empty-first-line': null,
        'number-max-precision': null,
        'property-no-vendor-prefix': null,
        'scss/at-rule-no-unknown': [
            true,
            {
                ignoreAtRules: [
                    'each',
                    'else',
                    'extends',
                    'for',
                    'function',
                    'if',
                    'ignores',
                    'include',
                    'media',
                    'mixin',
                    'return',
                    'use',

                    // Font Awesome 4
                    'fa-font-path',
                ],
            },
        ],
        'scss/double-slash-comment-empty-line-before': null,
        'scss/double-slash-comment-whitespace-inside': null,
        'selector-class-pattern': null,
        'shorthand-property-no-redundant-values': null,
    },
};
