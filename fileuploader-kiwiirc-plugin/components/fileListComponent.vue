<template>
    <div class="kiwi-filebuffer-outer-container">
        <div v-if="typeof fileList === 'undefined' || !fileList.length" class="kiwi-filebuffer-status-message">
            No new files have been uploaded!<br><br><br>
            Check back here to see the file history...
        </div>
        <div v-else class="kiwi-filebuffer-inner-container">
            Recently uploaded files...<br><br>
            <table class="kiwi-filebuffer-table">
                <tr><th></th><th>Nick</th><th>Link</th><th>Time</th></tr>
                <tr v-for="(file, idx) in fileList" :key="file">
                    <td class="kiwi-filebuffer-table-td">
                        <button
                            :href="file.url"
                            @click="fileList.splice(idx,1)"
                            class="kiwi-filebuffer-trash-button"
                        >
                            <i class="fa fa-trash" aria-hidden="true"/>
                        </button>
                    </td>
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
            fileList,
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
    margin: 10px;
    font-family: arial, tahoma;
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
.kiwi-filebuffer-trash-button {
    background: #f88;
    color: #822;
    border: none;
    border-radius: 5px;
    padding-left: 5px;
    padding-right: 5px;
    font-size: 18px;
    cursor: pointer;
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
