<template>
    <div class="kiwi-filebuffer-container">
        <div v-if="!bufferFiles.length" class="kiwi-filebuffer-empty">
            No files have recently been uploaded...
        </div>
        <template v-else>
            <div
                v-for="upload in bufferFiles"
                :key="upload.id"
                class="kiwi-filebuffer-card"
            >
                <div class="kiwi-filebuffer-details">
                    <a
                        :href="upload.url"
                        title="Preview File"
                        class="kiwi-filebuffer-title"
                        @click.prevent.stop="loadContent(upload.url)"
                    >
                        <span>{{ upload.name }}</span>
                        <span>{{ upload.ext }}</span>
                    </a>
                    <div class="kiwi-filebuffer-info kiwi-filebuffer-time">
                        {{ upload.expires || upload.time }}
                    </div>
                    <div class="kiwi-filebuffer-nicksize">
                        <span class="kiwi-filebuffer-info kiwi-filebuffer-nick">
                            {{ upload.nick }}
                        </span>
                        <span
                            v-if="upload.size"
                            class="kiwi-filebuffer-info kiwi-filebuffer-size"
                        >
                            {{ upload.size }}
                        </span>
                    </div>
                </div>
                <div class="kiwi-filebuffer-download">
                    <a
                        :href="upload.url"
                        title="Download File"
                        target="_blank"
                        download
                    >
                        <i class="fa fa-download kiwi-filebuffer-icon" />
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

const urlRegex = new RegExp(
    '/(?<id>[a-f0-9]{32})(?:/(?<name>.+?)(?<ext>\\.[^.\\s]+)?)?$',
);

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
                    console.error(
                        'failed to match fileuploader url',
                        msg.message,
                    );
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

<style scoped>
.kiwi-filebuffer-container {
    font-family: arial, tahoma;
    line-height: normal;
    height: 100%;
    width: 100%;
    box-sizing: border-box;
}

.kiwi-filebuffer-empty {
    box-sizing: border-box;
}

.kiwi-filebuffer-card {
    display: flex;
    width: 100%;
    padding: 6px;
    margin-bottom: 0.5em;
    box-sizing: border-box;
    background: #666;
    overflow: hidden;
    white-space: nowrap;
    color: #fff;
    line-height: normal;
}

.kiwi-filebuffer-card:last-of-type {
    margin-bottom: 0;
}

.kiwi-filebuffer-details {
    flex-grow: 1;
    width: 0;
}

.kiwi-filebuffer-details span {
    display: inline-block;
}

.kiwi-filebuffer-details > * {
    margin-bottom: 4px;
}

.kiwi-filebuffer-details > *:last-child {
    margin-bottom: 0;
}

.kiwi-filebuffer-details > *:not(:first-child) {
    font-size: 90%;
}

.kiwi-filebuffer-title {
    display: inline-flex;
    font-weight: bold;
    text-decoration: none;
    max-width: 100%;
    color: #42b992;
    cursor: pointer;
}

.kiwi-filebuffer-title > span:first-child {
    text-overflow: ellipsis;
    overflow-x: hidden;
}

.kiwi-filebuffer-time {
    text-overflow: ellipsis;
    overflow-x: hidden;
}

.kiwi-filebuffer-nicksize {
    display: flex;
}

.kiwi-filebuffer-nick {
    flex-grow: 1;
    text-overflow: ellipsis;
    overflow-x: hidden;
    margin-right: 10px;
}

.kiwi-filebuffer-download {
    display: flex;
    align-items: center;
    padding-left: 1em;
    padding-right: 0.2em;
}

.kiwi-filebuffer-icon {
    display: flex;
    width: 32px;
    height: 32px;
    font-size: 16px;
    align-items: center;
    justify-content: center;
    border: 2px solid #fff;
    border-radius: 50%;
}
</style>
