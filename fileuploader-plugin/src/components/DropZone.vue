<template>
  <div
    id="full-page-drop-zone"
    class="drop-zone"
    v-bind:class="{ 'drop-hover': isDropHovered }"
  >
    <span class="drop-message">Drop a file anywhere</span>
  </div>
</template>

<script>
/*
import tus from 'tus-js-client'

function unwrapSingle(list) {
  if (list.length !== 1) {
    throw new TypeError(`expected length 1, got ${list.length}`)
  }
  return list[0]
}
*/

// const tusOptions = {
//   endpoint: 'http://127.0.0.1:8088/files',
//   onError : function (error) {
//     debugger
//     if (error.originalRequest) {
//       if (window.confirm("Failed because: " + error + "\nDo you want to retry?")) {
//         upload.start();
//         uploadIsRunning = true;
//         return;
//       }
//     } else {
//       window.alert("Failed because: " + error);
//     }

//     // reset();
//   },
//   onProgress: function (bytesUploaded, bytesTotal) {
//     var percentage = (bytesUploaded / bytesTotal * 100).toFixed(2);
//     // progressBar.style.width = percentage + "%";
//     console.log(bytesUploaded, bytesTotal, percentage + "%");
//   },
//   onSuccess: function () {
//     console.log("success", arguments)
//     // debugger
//     // var anchor = document.createElement("a");
//     // anchor.textContent = "Download " + upload.file.name + " (" + upload.file.size + " bytes)";
//     // anchor.href = upload.url;
//     // anchor.className = "btn btn-success";
//     // uploadList.appendChild(anchor);

//     // reset();
//   }
// }

export default {
  name: 'DropZone',
  props: {},
  data: () => ({
    isDropHovered: false
  }),
  created () {
    window.addEventListener('dragenter', this.windowDragenter)
    // window.addEventListener('dragleave', this.windowDragleave)
  },
  destroyed () {
    window.removeEventListener('dragenter', this.windowDragenter)
    // window.removeEventListener('dragleave', this.windowDragleave)
  },
  methods: {
    windowDragenter (event) {
      this.isDropHovered = true
    },
    // windowDragleave(event) {
    //   this.isWindowDropHovered = false
    // },
    dragleave (event) {
      this.isDropHovered = false
    },
    drop (event) {
      this.isDropHovered = false
      /*
      const file = unwrapSingle(event.dataTransfer.files)
      // const upload = new tus.Upload(file, tusOptions)
      const upload = new tus.Upload(file, {
        endpoint: "http://localhost:8088/files",
        // retryDelays: [0, 1000, 3000, 5000],
        metadata: {
          filename: file.name,
          filetype: file.type
        },
        onError: function(error) {
          console.log("Failed because: " + error)
        },
        onProgress: function(bytesUploaded, bytesTotal) {
          var percentage = (bytesUploaded / bytesTotal * 100).toFixed(2)
          console.log(bytesUploaded, bytesTotal, percentage + "%")
        },
        onSuccess: function() {
          console.log("Download %s from %s", upload.file.name, upload.url)
        }
      })
      upload.start()
      */
    }
  }
}
</script>

<style>
html, body {
  min-height: 100%;
}
</style>

<style scoped>
.drop-zone {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  font-size: 5em;
  display: none;
  align-items: center;
  justify-content: center;
  background-color: rgba(192, 192, 192, 0.8);
}
.drop-zone * {
  /* prevents drags over child elements from triggering dragleave on the
  .drop-zone element. an alternative would be to use a counting semaphore
  controlled by enters and leaves at every level to hold the overlay enabled */
  pointer-events: none;
}
.drop-hover {
  display: flex;
}
</style>
