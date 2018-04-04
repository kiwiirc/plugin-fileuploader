import { Core as Uppy, Dashboard, Webcam, Tus } from 'uppy'
import 'uppy/dist/uppy.min.css'

kiwi.plugin('fileuploader', function (kiwi, log) {
	// add button to input bar
	const uploadFileButton = document.createElement('i')
	uploadFileButton.className = 'upload-file-button fa fa-upload'

	kiwi.addUi('input', uploadFileButton)

	const uppy = Uppy({
		autoProceed: false,
		onBeforeFileAdded: (currentFile, files) => {
			const buffer = kiwi.state.getActiveBuffer()
			const isValidTarget = buffer.isChannel() || buffer.isQuery()
			if (!isValidTarget) {
				return Promise.reject('Files can only be shared in channels or queries.')
			}
			return Promise.resolve()
		},
		restrictions: {
			maxFileSize: 10485760, // 10 MiB
		},
	})
		.use(Dashboard, { trigger: uploadFileButton })
		.use(Webcam, { target: Dashboard })
		.use(Tus, { endpoint: 'http://localhost:8088/files' })
		.run()

	// show uppy modal whenever a file is dragged over the page
	window.addEventListener('dragenter', event => {
		uppy.getPlugin('Dashboard').openModal()
	})

	uppy.on('upload-success', (file, resp, uploadURL) => {
		// append filename to uploadURL
		uploadURL = `${uploadURL}/${encodeURIComponent(file.name)}`
		uppy.setFileState(file.id, { uploadURL })

		// send a message with the url of each successful upload
		kiwi.emit('input.raw', uploadURL)
	})

	uppy.on('complete', result => {
		// automatically close upload modal if all uploads succeeded
		if (result.failed.length === 0) {
			// TODO: this would be nicer with a css transition: delay, then fade out
			uppy.getPlugin('Dashboard').closeModal()
		}
	})
})
