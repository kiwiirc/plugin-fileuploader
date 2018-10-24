<template>
    <div class="kiwi-filebuffer-outer-container">
        <div v-if="typeof fileList === 'undefined' || !fileList.length" class="kiwi-filebuffer-status-message">
            No files have recently been <br>
            uploaded to this channel {{currentBuffer}}
        </div>
        <div v-else class="kiwi-filebuffer-inner-container">
            <span class="kiwi-filebuffer-status-message">Files recently uploaded to {{currentBuffer}}</span><br><br>
            <table class="kiwi-filebuffer-table">
                <tr><th>Nick</th><th>Link</th><th>Time</th></tr>
                <tr v-for="(file, idx) in fileList" :key="file">
                    <td class="kiwi-filebuffer-table-td">
                        {{file.nick}}
                    </td>
                    <td class="kiwi-filebuffer-table-td">
                        <a
                            :href="file.url"
                            @click.prevent.stop="loadContent(file.url)"
                            class="kiwi-filebuffer-anchor"
                        >
                            {{getFileName(file.url)}}
                        </a>
                    </td>
                    <td class="kiwi-filebuffer-table-td">
                        {{file.time}}
                    </td>
                </tr>
            </table>
        </div>
    </div>
</template>

<script>

'kiwi public';

export default {
    data() {
        return {
            fileList: [],
            settings: kiwi.state.setting('fileuploader'),
        };
    },
    methods: {
        getFileName(file) {
            file = decodeURI(file);
            let name = file.split('/')[file.split('/').length-1];
            if (name.length >= 25) {
                name = name.substring(0, 18) + '...' + name.substring(name.length - 4);
            }
            return name;
        },
        loadContent(url) {
            kiwi.emit('mediaviewer.show', url);
        },
        messageHandler() {
            setTimeout(() => this.sharedFiles(kiwi.state.getActiveBuffer()), 1100)
        },
        pruneDups(a) {
            for(let i = 0; i < a.length; ++i) {
                for(let j = i + 1; j < a.length; ++j) {
                    if(a[i].url === a[j].url) a.splice(j, 1)
                }
            }
        },
        sharedFiles(buffer) {
            this.fileList = []
            for(let i = 0; i < buffer.messagesObj.messages.length; i++) {
                let e = buffer.messagesObj.messages[i]
                if (e.message.indexOf(this.settings.server) !== -1) {
                    let currentdate = new Date(e.time);
                    let time = ("00" + currentdate.getHours()).slice(-2) + ":"
                        + ("00" + currentdate.getMinutes()).slice(-2) + ":"
                        + ("00" + currentdate.getSeconds()).slice(-2)
                    let link = {
                        url: e.message.substring(e.message.indexOf(this.settings.server)),
                        nick: e.nick,
                        time
                    };
                    link.url = link.url.split(' ')[0].split(')')[0]
                    this.fileList.push(link);
                }
            }
            // comment out the following line to include duplicates
            this.pruneDups(this.fileList)
            kiwi.emit('files.listshared', { fileList: this.fileList, buffer })
            return this.fileList
        },
    },
    computed: {
        currentBuffer() {
            this.sharedFiles(kiwi.state.getActiveBuffer())
            return kiwi.state.getActiveBuffer().name
        },
    },
    destroyed() {
        kiwi.off('message.new', this.messageHandler)
    },
    mounted() {
        this.sharedFiles(kiwi.state.getActiveBuffer())
        kiwi.on('message.new', this.messageHandler)
    }
}
</script>

<style scoped>
.kiwi-filebuffer-outer-container {
    overflow: auto;
    background: #eee;
    height: 100%;
}
.kiwi-filebuffer-status-message {
    margin: 10px;
    font-family: arial, tahoma;
    font-size: 18px;
}
.kiwi-filebuffer-inner-container {
    margin: 10px;
    font-family: arial, tahoma;
}
.kiwi-filebuffer-table {
    width: 100%;
    border-collapse: collapse;
}
.kiwi-filebuffer-table-td {
    border-bottom: 1px solid #aaa;
    text-align: center;
}
.kiwi-filebuffer-anchor {
    display: inline-block;
    text-align: center;
    width: 100%;
    border: none;
    background: #afa;
    color: #006;
    line-height: 1.1em;
    border-radius: 5px;
    text-decoration: underline;
    cursor: pointer;
}
</style>
