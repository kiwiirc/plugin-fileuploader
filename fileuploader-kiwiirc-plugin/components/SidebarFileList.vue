<template>
    <div class="kiwi-filebuffer-outer-container">
        <div v-if="!fileList.length">
            No files have recently been uploaded...
        </div>
        <div v-else class="kiwi-filebuffer-inner-container">
            <div v-for="(file, idx) in fileList.slice().reverse()" :key="file" class="kiwi-filebuffer-download-container">
                <a
                    :href="file.url"
                    class="kiwi-filebuffer-anchor"
                    download
                    title="Download File"
                >
                    <i class="fa fa-download kiwi-filebuffer-downloadicon"/>
                </a>
                <div class="kiwi-filebuffer-filetitle" style="font-size: 18px;">
                    <a
                        :href="file.url"
                        @click.prevent.stop="loadContent(file.url)"
                        class="kiwi-filebuffer-anchor"
                        title="Preview File"
                    >
                        {{ file.fileName }}
                    </a>
                </div>
                <div class="kiwi-filebuffer-fileinfo"> {{ file.nick }}</div>
                <div class="kiwi-filebuffer-fileinfo"> {{ file.time }}</div>
                <div style="clear: both;"></div>
            </div>
        </div>
    </div>
</template>

<script>

'kiwi public';

export default {
    data() {
        return {
            settings: kiwi.state.setting('fileuploader'),
        };
    },
    methods: {
        getFileName(file) {
            file = decodeURI(file);
            let name = file.split('/')[file.split('/').length-1];
            if (name.length >= 20) {
                name = name.substring(0, 13) + '...' + name.substring(name.length - 4);
            }
            return name;
        },
        truncateNick(nick) {
            if (nick.length >= 13) {
                nick = nick.substring(0, 11) + "\u2026";
            }
            return nick;
        },
        loadContent(url) {
            kiwi.emit('mediaviewer.show', url);
        },
        sharedFiles(buffer) {
            let returnArr = []
            if(buffer === null) return [];
            let messages = buffer.getMessages()
            let tmp = buffer.message_count
            for(let i = 0; i < messages.length; i++) {
                let e = messages[i]
                if (e.message.indexOf(this.settings.server) !== -1) {
                    let time = new Intl.DateTimeFormat('default', { hour: 'numeric', minute: 'numeric', second: 'numeric' }).format(new Date(e.time))
                    let url = e.message.substring(e.message.indexOf(this.settings.server)).split(' ')[0].split(')')[0].split('\'')[0]
                    let link = {
                        url,
                        nick: this.truncateNick(e.nick),
                        fileName: this.getFileName(url),
                        time
                    };
                    returnArr.push(link);
                }
            }

            // comment out the following line to include duplicates
            returnArr = _.uniqBy(returnArr, 'url')

            kiwi.emit('files.listshared', { fileList: returnArr, buffer })
            return returnArr
        },
    },
    computed: {
        fileList() {
            return this.sharedFiles(kiwi.state.getActiveBuffer()) 
        }
    },
}
</script>

<style scoped>
.kiwi-filebuffer-outer-container {
    height: 100%;
    width: 100%;
    margin-top: 5px;
}
.kiwi-filebuffer-inner-container {
    font-family: arial, tahoma;
    width: 100%;
}
.kiwi-filebuffer-download-container {
    margin: 0 0 5px 0;
    padding: 10px 10px;
    background: #666;
    color: #eee;
}
.kiwi-filebuffer-downloadicon {
    float: right;
    margin-top: 3px;
    margin-right: 3px;
    border: 1px solid #fff;
    border-radius: 50%;
    padding: 7px;
    font-size: 16px;
}
.kiwi-filebuffer-filetitle {
    font-weight: bold;
    line-height: normal;
    margin-bottom: 5px;
}
.kiwi-filebuffer-fileinfo {
    float: left;
    font-size: 11px;
    width: 90px;
    opacity: 0.8;
    line-height: normal;
}
.kiwi-filebuffer-anchor {
    border: none;
    color: #eee;
    cursor: pointer;
    text-decoration: none;
}
</style>
