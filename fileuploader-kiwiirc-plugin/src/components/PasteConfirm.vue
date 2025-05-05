<template>
    <div class="p-fileuploader-paste-confirm" @click.prevent.stop @mousedown.prevent.stop>
        <span>
            You pasted a lot of text.<br />
            Would you like to upload as a file instead?
        </span>
        <input-confirm
            :flip-connotation="true"
            @ok="uploadInput()"
            @cancel="sendInput()"
        />
    </div>
</template>

<script>
/* global kiwi:true */

export default {
    beforeDestroy() {

    },
    methods: {
        uploadInput() {

        },
        sendInput() {
            kiwi.fileuploader.allowPaste = true;
            document.execCommand('paste');
            // if (kiwi.fileuploader.pasteEvent) {
            //     const oldEvent = kiwi.fileuploader.pasteEvent;
            //     const newEvent = new ClipboardEvent(oldEvent.type, oldEvent);
            //     newEvent.fileuploaderApproved = true;
            //     newEvent.target = oldEvent.target;
            //     // newEvent.srcElement = oldEvent.srcElement;
            //     // newEvent.clipboardData = oldEvent.clipboardData;
            //     delete kiwi.fileuploader.pasteEvent;
            //     console.log('newEvent', newEvent);
            //     document.dispatchEvent(newEvent);
            //     document.execCommand('paste');
            // }
            this.$state.$emit('input.tool', null);
        },
    },
};
</script>

<style lang="scss">
.p-fileuploader-paste-confirm {
    position: absolute;
    bottom: 0;
    left: 50%;
    z-index: 1;
    padding: 6px 10px 10px 10px;
    border: 1px solid var(--comp-border, #b2b2b2);
    border-bottom: 0;
    border-radius: 10px 10px 0 0;
    transform: translateX(-50%);

    .u-input-confirm {
        padding: initial;
        padding-left: 10px;
    }
}
</style>
