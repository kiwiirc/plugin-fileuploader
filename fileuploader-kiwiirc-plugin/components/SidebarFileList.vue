<template>
    <div class="kiwi-filebuffer-outer-container">
        <div v-if="!fileList.length" class="kiwi-filebuffer-status-message">
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
                            {{file.fileName}}
                        </a>
                    </td>
                    <td class="kiwi-filebuffer-table-td kiwi-filebuffer-time">
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
        sharedFiles(buffer) {
            let returnArr = []
            let messages = buffer.getMessages()
            let tmp = buffer.message_count
            for(let i = 0; i < messages.length; i++) {
                let e = messages[i]
                if (e.message.indexOf(this.settings.server) !== -1) {
                    let time = new Intl.DateTimeFormat('default', { hour: 'numeric', minute: 'numeric', second: 'numeric' }).format(new Date(e.time))
                    let url = e.message.substring(e.message.indexOf(this.settings.server))
                    let link = {
                        url,
                        nick: e.nick,
                        fileName: this.getFileName(url),
                        time
                    };
                    link.url = link.url.split(' ')[0].split(')')[0].split('\'')[0]
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
        currentBuffer() {
            return kiwi.state.getActiveBuffer().name
        },
        fileList() {
            return this.sharedFiles(kiwi.state.getActiveBuffer()) 
        }
    },
}
</script>

<style scoped>
.kiwi-filebuffer-outer-container {
    overflow: auto;
    background: #eee;
    height: 100%;
}
.kiwi-filebuffer-status-message {
    margin: 20px;
    font-family: arial, tahoma;
    font-size: 18px;
}
.kiwi-filebuffer-inner-container {
    margin: 5px;
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
.kiwi-filebuffer-time {
    font-size: 0.95em;
}
</style>
