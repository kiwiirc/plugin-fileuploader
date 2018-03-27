import Uppy from 'uppy/lib/core'
import Dashboard from 'uppy/lib/plugins/Dashboard'
import DragDrop from 'uppy/lib/plugins/DragDrop'
import Tus from 'uppy/lib/plugins/Tus'
import 'uppy/dist/uppy.min.css'
import DropZone from './DropZone.vue'
import Vue from 'vue'

kiwi.plugin('fileuploader', function(kiwi, log) {
	// load stylesheet
	const styles = document.createElement('link')
	styles.rel = 'stylesheet'
	styles.href = 'http://localhost:1234/dist/fileuploader-kiwiirc-plugin.css'
	document.head.appendChild(styles)

	// add button to input bar
	const uploadFileButton = document.createElement('i')
	uploadFileButton.className = 'upload-file-button fa fa-upload'

	const dropArea = document.createElement('div')
	dropArea.className = 'drop-area'

	uploadFileButton.appendChild(dropArea)

	new Vue({
		render: createElement => createElement(DropZone)
	}).$mount(dropArea)

	kiwi.addUi('input', uploadFileButton)

	//
	const uppy = Uppy({ autoProceed: false })
		.use(Dashboard, {
			trigger: uploadFileButton
		})
		.use(DragDrop, {
			target: dropArea
		})
		.use(Tus, { endpoint: 'http://localhost:8088/files' })
		.run()

	// set up full-page drop-zone
	const dragState = {
		count: 0
	}
	document.addEventListener('dragenter', event => {
		document.body.classList.add('drag-hover')
		dragState.count += 1
	})

	const dropZone = document.createElement('div')
	dropZone.className = 'drop-zone'

	dropZone.addEventListener

	const dropZoneMessage = document.createElement('span')
	dropZoneMessage.className = 'drop-message'
	dropZoneMessage.innerText = 'Drop a file anywhere'
	dropZone.appendChild(dropZoneMessage)


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
