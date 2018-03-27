import Uppy from 'uppy/lib/core'
import Dashboard from 'uppy/lib/plugins/Dashboard'
// import DragDrop from 'uppy/lib/plugins/DragDrop'
import Tus from 'uppy/lib/plugins/Tus'
import 'uppy/dist/uppy.min.css'
// import DropZone from './components/DropZone.vue'
// import Vue from 'vue'
/* global kiwi */

/* function querySelectorOnly (rootElement, query) {
  const nodes = rootElement.querySelectorAll(query)
  if (nodes.length !== 1) {
    throw new TypeError(`Expected single QuerySelector match, got ${nodes.length} matches`)
  }
  return nodes[0]
} */

kiwi.plugin('fileuploader', function (kiwi, log) {
  // load stylesheet manually
  // const styles = document.createElement('link')
  // styles.rel = 'stylesheet'
  // styles.href = 'http://localhost:1234/dist/fileuploader-kiwiirc-plugin.css'
  // document.head.appendChild(styles)

  // add button to input bar
  const uploadFileButton = document.createElement('i')
  uploadFileButton.className = 'upload-file-button fa fa-upload'

  // const dropArea = document.createElement('div')
  // dropArea.className = 'drop-area'

  // uploadFileButton.appendChild(dropArea)

  // new Vue({
  //   render: createElement => createElement(DropZone)
  // }).$mount(dropArea)

  kiwi.addUi('input', uploadFileButton)

  const uppy = Uppy({ autoProceed: false })
    .use(Dashboard, {
      trigger: uploadFileButton
    })
    // .use(DragDrop, {
    //   target: document.querySelector('#full-page-drop-zone')
    // })
    .use(Tus, { endpoint: 'http://localhost:8088/files' })
    .run()

  // show uppy modal whenever a file is dragged over the page
  window.addEventListener('dragenter', event => {
    uppy.getPlugin('Dashboard').openModal()
  })

  uppy.on('upload-success', (file, resp, uploadURL) => {
    /*
    const editor = querySelectorOnly(document, '.kiwi-ircinput-editor')

    // append new urls to the current contents of the ircinput editor with space as separator
    editor.innerText = [editor.innerText, uploadURL].filter(s => s !== '').join(' ')
    */

    // send a message with the url of each successful upload
    kiwi.emit('input.raw', uploadURL)
  })

  uppy.on('complete', result => {
    // automatically close upload modal if all uploads succeeded
    if (result.failed.length === 0) {
      uppy.getPlugin('Dashboard').closeModal()
      // TODO: this would be nicer with a css transition: delay, then fade out
    }
    /*
      // set focus at the end of the input if we appended urls
      if (!(result.successful.length > 0)) {
        return
      }
      const editor = querySelectorOnly(document, '.kiwi-ircinput-editor')
      const range = document.createRange()
      const inputTextNode = editor.childNodes[0]
      const offset = inputTextNode.length
      range.setStart(inputTextNode, offset)
      range.setEnd(inputTextNode, offset)
      const selection = window.getSelection()
      selection.removeAllRanges()
      selection.addRange(range)
      // not working...
      inputTextNode.scrollIntoView({
        block: 'end'
      })

    } */
  })

  // set up full-page drop-zone
  // const dragState = {
  //   count: 0
  // }
  // document.addEventListener('dragenter', event => {
  //   document.body.classList.add('drag-hover')
  //   dragState.count += 1
  // })

  // const dropZone = document.createElement('div')
  // dropZone.className = 'drop-zone'

  // dropZone.addEventListener

  // const dropZoneMessage = document.createElement('span')
  // dropZoneMessage.className = 'drop-message'
  // dropZoneMessage.innerText = 'Drop a file anywhere'
  // dropZone.appendChild(dropZoneMessage)

  // debugger // try to find ref to Dashboard instance .openModal method
  // uppy.getPlugin('Dashboard').openModal()
})

/*
const Uppy = require('uppy/lib/core')
const Dashboard = require('uppy/lib/plugins/Dashboard')
const Tus = require('uppy/lib/plugins/Tus')

const uppy = Uppy({ autoProceed: false })
  .use(Dashboard, {
    trigger: '#select-files'
  })
  .use(Tus, {endpoint: 'https://master.tus.io/files/'})
  .run()

uppy.on('complete', (result) => {
  console.log(`Upload complete! We’ve uploaded these files: ${result.successful}`)
})
*/
