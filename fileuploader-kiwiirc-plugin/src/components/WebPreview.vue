<template>
    <div
        v-if="error"
        class="kiwi-webpreview-error"
        :class="{'kiwi-mediaviewer-center': !showPin}"
    >{{ error }}</div>
    <iframe
        v-else
        ref="previewFrame"
        :sandbox="iframeSandboxOptions"
        frameborder="0"
        width="100%"
        class="kiwi-webpreview-frame"
    />
</template>

<script>
'kiwi public';

let embedlyTagIncluded = false;

export default {
    props: ['url', 'showPin', 'iframeSandboxOptions'],
    data() {
        return {
            error: '',
            eventListener: null,
            debouncedUpdateEmbed: null,
        };
    },
    computed: {
        settings() {
            return this.$state.setting('fileuploader.webpreview');
        },
    },
    watch: {
        url() {
            this.updateEmbed();
        },
    },
    mounted() {
        console.log('mounted');
        this.updateEmbed();
    },
    methods: {
        updateEmbed() {
            console.log('updateEmbed');
            const iframe = this.$refs.previewFrame;
            if (!iframe) {
                // No iframe to work with so nothing to update
                return;
            }

            let newUrl = this.settings.url
                .replace('{url}', encodeURIComponent(this.url))
                .replace('{center}', !this.showPin)
                .replace('{width}', this.settings.maxWidth || 1000)
                .replace('{height}', this.settings.maxHeight || 400);

            // Set the iframe url
            iframe.src = newUrl;

            clearTimeout();

            this.iframeTimeout = this.setTimeout(() => {
                this.$emit('setHeight', 'auto');
                this.$emit('setMaxHeight', '40%');
                this.error = this.$t('preview_failed');
            }, 2000);

            // Add message event listener if it does not exist
            this.maybeAddOrRemoveEventListener(true);
        },
        messageEventHandler(event) {
            console.log('messageEventHandler');
            const iframe = this.$refs.previewFrame;
            if (!iframe || event.source !== iframe.contentWindow) {
                // The message event did not come from our iframe ignore it
                return;
            }

            console.log('messageEventHandler', event.data);

            const data = event.data;
            if (data.error) {
                // Error message indicates the url cannot be embedded
                this.error = (data.error === 'not_supported') ?
                    this.$t('preview_not_supported') :
                    data.error;

                // stop hiding the media viewer to show the error
                this.$emit('setHeight', 'auto');
                this.$emit('setMaxHeight', '40%');
            } else if (data.dimensions) {
                // Dimensions message contains updated dimensions for the iframe content

                const height = this.showPin ?
                    Math.min(data.dimensions.height, this.settings.maxHeight || 400)
                    : data.dimensions.height;
                iframe.height = height + 'px';

                // Now we have dimensions stop hiding the media viewer
                // This is to stop the message list jumping when opened with invalid url
                this.$emit('setHeight', 'auto');
                this.$emit('setMaxHeight', '40%');
            }

            if (data.error || data.dimensions) {
                this.clearTimeout();
            }
        },
        maybeAddOrRemoveEventListener(add) {
            if (add && !this.eventListener) {
                this.eventListener = this.listen(window, 'message', this.messageEventHandler);
            } else if (!add && this.eventListener) {
                // Remove event listener
                this.eventListener();
                this.eventListener = null;
            }
        },
        clearTimeout() {
            if (!this.iframeTimeout) {
                return;
            }
            clearTimeout(this.iframeTimeout);
            this.iframeTimeout = 0;
        }
    },
};

</script>

<style>

    .kiwi-webpreview-error {
        display: inline-block;
        padding: 10px 15px;
        border-radius: 10px;
        border: 1px solid var(--brand-midtone);
    }

    .kiwi-mediaviewer--pinned .kiwi-webpreview-frame {
        display: block;
        max-height: initial;
    }

</style>
