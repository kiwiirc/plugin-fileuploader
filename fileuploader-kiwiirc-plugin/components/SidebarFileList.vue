<template>
    <div class="kiwi-filebuffer-outer-container">
        <div v-if="!fileList.length">
            No files have recently been uploaded...
        </div>
        <div v-else class="kiwi-filebuffer-inner-container">
            <div v-for="(file, idx) in fileList.slice().reverse()" :key="file" class="kiwi-filebuffer-download-container">
                <a
                    :href="file.url"
                    @click.prevent.stop="loadContent(file.url)"
                    class="kiwi-filebuffer-anchor"
                >
                    <i class="fa fa-download" style="float: right; font-size: 40px; margin-top:4px; margin-right: 3px;"/>
                    <div style="font-size: 18px;">{{ file.fileName }}</div>
                    <div style="font-size: 11px;"> {{ file.nick }} &nbsp;&nbsp; {{ file.time }}</div>
                </a>
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
            if (nick.length >= 10) {
                nick = nick.substring(0, 8) + "\u2026";
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
    overflow: auto;
    height: 100%;
    width: 100%;
}
.kiwi-filebuffer-inner-container {
    font-family: arial, tahoma;
    width: 100%;
}
.kiwi-filebuffer-download-container {
    margin: 10px;
    border-radius: 5px;
    padding: 5px;
    padding-bottom: 0;
    background: #666;
    color: #eee;
}
.kiwi-filebuffer-anchor {
    border: none;
    color: #eee;
    cursor: pointer;
    text-decoration: none;
}
</style>
