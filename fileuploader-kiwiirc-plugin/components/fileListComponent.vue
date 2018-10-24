<template>
    <div class="kiwi-filebuffer-outer-container">
        <div v-if="typeof fileList === 'undefined' || !fileList.length" class="kiwi-filebuffer-status-message">
            No files have recently been <br>
            uploaded to this channel<br>
            ({{currentBufferName}})
        </div>
        <div v-else class="kiwi-filebuffer-inner-container">
            <span class="kiwi-filebuffer-status-message">Files recently uploaded to:<br>{{currentBuffer}} ...</span><br><br>
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
            let name = file.split('/')[file.split('/').length-1];
            if (name.length >= 25) {
                name = name.substring(0, 18) + '...' + name.substring(name.length - 4);
            }
            return name;
        },
        loadContent(url) {
            kiwi.emit('mediaviewer.show', url);
        },
        getFileList() {
            this.fileList = []
            let buffer = kiwi.state.getActiveBuffer()
            let parse = (e) => {
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
            for(let i = 0; i < buffer.messagesObj.messages.length; i++) {
                parse(buffer.messagesObj.messages[i])
            }
            kiwi.emit('files.listshared', { fileList: this.fileList, buffer })
            return this.fileList
        }
    },
    computed: {
        currentBufferName() {
            this.getFileList()
            return kiwi.state.getActiveBuffer().name
        }
    },
    destroyed() {
        kiwi.off('message.new', () => { this.getFileList() })
    },
    mounted() {
        this.getFileList()
        kiwi.on('message.new', () => { this.getFileList() })
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
    background: #8f8;
     border-radius: 5px;
    text-decoration: underline;
    cursor: pointer;
}
</style>
