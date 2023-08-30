<template>
    <div class="kiwiirc-fileuploader-body">
        <div class="kiwi-messagelist-body" v-html="ml.formatMessage(message)" />
        <div v-if="shouldShow" class="kiwiirc-fileuploader-audio">
            <audio
                :src="message.mentioned_urls[0]"
                preload="metadata"
                v-bind="flags"
                @loadedmetadata="hasMeta = true"
                @error="onError"
            />
        </div>
    </div>
</template>

<script>
export default {
    props: ['ml', 'message'],
    data() {
        return {
            hasMeta: false,
        };
    },
    computed: {
        shouldShow() {
            if (!this.message.mentioned_urls.length) {
                return false;
            }
            if (this.message.fileuploader.hasError) {
                return false;
            }
            return true;
        },
        flags() {
            if (!this.hasMeta) {
                return {};
            }
            return { controls: 'controls' };
        },
    },
    methods: {
        onError() {
            this.message.bodyTemplate = null;
            this.message.fileuploader.hasError = true;
        },
    },
};
</script>
