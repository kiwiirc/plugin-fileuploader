<template>
    <div class="p-fileuploader-sidebar-container">
        <div v-if="!bufferFiles.length" class="p-fileuploader-sidebar-empty">
            No files have recently been uploaded...
        </div>
        <template v-else>
            <div
                v-for="upload in bufferFiles"
                :key="upload.id"
                class="p-fileuploader-sidebar-card"
            >
                <div class="p-fileuploader-sidebar-details">
                    <a
                        :href="upload.url"
                        title="Preview File"
                        class="p-fileuploader-sidebar-title"
                        @click.prevent.stop="loadContent(upload.url)"
                    >
                        <span>{{ upload.name }}</span>
                        <span>{{ upload.ext }}</span>
                    </a>
                    <div class="p-fileuploader-sidebar-info p-fileuploader-sidebar-time">
                        {{ upload.expires || upload.time }}
                    </div>
                    <div class="p-fileuploader-sidebar-nicksize">
                        <span class="p-fileuploader-sidebar-info p-fileuploader-sidebar-nick">
                            {{ upload.nick }}
                        </span>
                        <span
                            v-if="upload.size"
                            class="p-fileuploader-sidebar-info p-fileuploader-sidebar-size"
                        >
                            {{ upload.size }}
                        </span>
                    </div>
                </div>
                <div class="p-fileuploader-sidebar-download">
                    <a
                        :href="upload.url"
                        title="Download File"
                        target="_blank"
                        download
                    >
                        <i class="fa fa-download p-fileuploader-sidebar-icon" />
                    </a>
                </div>
            </div>
        </template>
    </div>
</template>

<script>
'kiwi public';

/* global _:true */
/* global kiwi:true */

import { bytesReadable, durationReadable } from '@/utils/readable';
import * as config from '@/config.js';

const urlRegex = /\/(?<id>[a-f0-9]{32})(?:\/(?<name>.+?)(?<ext>\.[^.\s]+)?)?$/;

export default {
    data() {
        return {
            messageWatcher: null,
            bufferFiles: [],
            updateFilesDebounced: null,
            hasExpires: false,
            expiresUpdater: 0,
        };
    },
    computed: {
        buffer() {
            return kiwi.state.getActiveBuffer();
        },
    },
    watch: {
        buffer() {
            if (this.messageWatcher) {
                this.messageWatcher();
                this.messageWatcher = null;
            }
            if (this.expiresUpdater) {
                clearTimeout(this.expiresUpdater);
                this.expiresUpdater = 0;
            }
            this.updateFiles();
        },
    },
    created() {
        this.updateFilesDebounced = _.debounce(this.updateFiles, 1000);
    },
    mounted() {
        this.updateFiles();
    },
    beforeDestroy() {
        if (this.messageWatcher) {
            this.messageWatcher();
            this.messageWatcher = null;
        }
        if (this.expiresUpdater) {
            clearTimeout(this.expiresUpdater);
            this.expiresUpdater = 0;
        }
    },
    methods: {
        loadContent(url) {
            kiwi.emit('mediaviewer.show', url);
        },
        updateFiles() {
            const files = [];
            const srvUrl = config.getSetting('server');
            const messages = this.buffer.getMessages();
            const existingIds = [];

            for (let i = 0; i < messages.length; i++) {
                const msg = messages[i];

                const upload = msg.fileuploader;
                const info = upload?.fileInfo;

                if (upload?.hasError) {
                    continue;
                }

                const url = msg.mentioned_urls[0];
                if (!url || (!upload && url.indexOf(srvUrl) !== 0)) {
                    continue;
                }

                if (info?.expires && info.expires < Date.now() / 1000) {
                    // upload already expired
                    continue;
                }

                const match = urlRegex.exec(url);
                if (!match) {
                    continue;
                }

                const details = {
                    id: match.groups.id,
                    url: url,
                    name: match.groups.name || '',
                    ext: match.groups.ext || '',
                    type: info?.type || '',
                    size: info?.size ? bytesReadable(info.size) : 0,
                    nick: msg.nick,
                    time: new Intl.DateTimeFormat('default', {
                        day: 'numeric',
                        month: 'numeric',
                        year: 'numeric',
                        hour: 'numeric',
                        minute: 'numeric',
                        second: 'numeric',
                    }).format(new Date(msg.time)),
                    expires: '',
                    expiresUnix: info?.expires || 0,
                };

                if (details.expiresUnix) {
                    this.hasExpires = true;
                }

                if (!existingIds.includes(details.id)) {
                    existingIds.push(details.id);
                    files.unshift(details);
                }
            }
            existingIds.length = 0;

            if (this.hasExpires) {
                this.updateExpires(files);
            }

            if (this.messageWatcher === null) {
                this.messageWatcher = this.$watch(
                    'buffer.message_count',
                    () => {
                        this.updateFilesDebounced();
                    },
                );
            }

            kiwi.emit('files.listshared', {
                fileList: files,
                buffer: this.buffer,
            });

            this.bufferFiles = files;
        },
        updateExpires(_files) {
            const files = _files || this.bufferFiles;
            if (this.expiresUpdater) {
                clearTimeout(this.expiresUpdater);
                this.expiresUpdater = 0;
            }
            for (let i = files.length - 1; i >= 0; i--) {
                const upload = files[i];
                if (!upload.expiresUnix) {
                    continue;
                }
                if (upload.expiresUnix < Date.now() / 1000) {
                    files.splice(i, 1);
                    continue;
                }
                upload.expires = durationReadable(
                    upload.expiresUnix - Math.floor(Date.now() / 1000),
                );
            }
            this.expiresUpdater = setTimeout(() => this.updateExpires(), 10000);
        },
    },
};
</script>

<style>
.p-fileuploader-sidebar-container {
    box-sizing: border-box;
    width: 100%;
    height: 100%;
    line-height: normal;
}

.p-fileuploader-sidebar-empty {
    box-sizing: border-box;
}

.p-fileuploader-sidebar-card {
    box-sizing: border-box;
    display: flex;
    width: 100%;
    padding: 6px;
    margin-bottom: 0.5em;
    overflow: hidden;
    line-height: normal;
    color: #fff;
    white-space: nowrap;
    background: #666;
}

.p-fileuploader-sidebar-card:last-of-type {
    margin-bottom: 0;
}

.p-fileuploader-sidebar-details {
    flex-grow: 1;
    width: 0;
}

.p-fileuploader-sidebar-details span {
    display: inline-block;
}

.p-fileuploader-sidebar-details > * {
    margin-bottom: 4px;
}

.p-fileuploader-sidebar-details > *:last-child {
    margin-bottom: 0;
}

.p-fileuploader-sidebar-details > *:not(:first-child) {
    font-size: 90%;
}

.p-fileuploader-sidebar-title {
    display: inline-flex;
    max-width: 100%;
    font-weight: bold;
    color: #42b992;
    text-decoration: none;
    cursor: pointer;
}

.p-fileuploader-sidebar-title > span:first-child {
    overflow-x: hidden;
    text-overflow: ellipsis;
}

.p-fileuploader-sidebar-time {
    overflow-x: hidden;
    text-overflow: ellipsis;
}

.p-fileuploader-sidebar-nicksize {
    display: flex;
}

.p-fileuploader-sidebar-nick {
    flex-grow: 1;
    margin-right: 10px;
    overflow-x: hidden;
    text-overflow: ellipsis;
}

.p-fileuploader-sidebar-download {
    display: flex;
    align-items: center;
    padding-right: 0.2em;
    padding-left: 1em;
}

.p-fileuploader-sidebar-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    font-size: 16px;
    border: 2px solid #fff;
    border-radius: 50%;
}
</style>
