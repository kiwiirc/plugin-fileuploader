<template>
    <div class="kiwiirc-fileuploader-body">
        <div class="kiwi-messagelist-body" v-html="ml.formatMessage(message)" />
        <div v-if="message.mentioned_urls.length" class="kiwiirc-fileuploader-audio">
            <audio :src="message.mentioned_urls[0]" preload="metadata" controls />
        </div>
    </div>
</template>

<script>
export default {
    props: ['ml', 'message'],
};
</script>
