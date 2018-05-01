import { Core as Uppy, Dashboard, Webcam, Tus } from 'uppy'
import 'uppy/dist/uppy.min.css'

const GB = 2**30

kiwi.plugin('fileuploader', function (kiwi, log) {
	// add button to input bar
	const uploadFileButton = document.createElement('i')
	uploadFileButton.className = 'upload-file-button fa fa-upload'

	kiwi.addUi('input', uploadFileButton)

	const uppy = Uppy({
		autoProceed: false,
		onBeforeFileAdded: (currentFile, files) => {
			const buffer = kiwi.state.getActiveBuffer()
			const isValidTarget = buffer && (buffer.isChannel() || buffer.isQuery())
			if (!isValidTarget) {
				return Promise.reject(new Error('Files can only be shared in channels or queries.'))
			}
			return Promise.resolve()
		},
		restrictions: {
			maxFileSize: kiwi.state.setting('fileuploader.maxFileSize') || 10*GB,
		},
	})
		.use(Dashboard, { trigger: uploadFileButton })
		.use(Webcam, { target: Dashboard })
		.use(Tus, { endpoint: kiwi.state.setting('fileuploader.server') || '/files' })
		.run()

	const dashboard = uppy.getPlugin('Dashboard')

	// show uppy modal whenever a file is dragged over the page
	window.addEventListener('dragenter', event => {
		dashboard.openModal()
	})

	// show uppy modal when files are pasted
	kiwi.on('buffer.paste', event => {
		const { files } = event.clipboardData

		// ensure a file has been pasted
		if (files.length <= 0) {
			return
		}

		// pass event to the dashboard for handling
		dashboard.openModal()
		dashboard.handlePaste(event)
	})

	uppy.on('upload-success', (file, resp, uploadURL) => {
		// append filename to uploadURL
		uploadURL = `${uploadURL}/${encodeURIComponent(file.meta.name)}`
		uppy.setFileState(file.id, { uploadURL })
		file = uppy.getFile(file.id)

		// emit a global kiwi event
		kiwi.emit('fileuploader.uploaded', { url: uploadURL, file: file })

		// send a message with the url of each successful upload
		kiwi.emit('input.raw', uploadURL)
	})

	uppy.on('complete', result => {
		// automatically close upload modal if all uploads succeeded
		if (result.failed.length === 0) {
			uppy.reset()
			// TODO: this would be nicer with a css transition: delay, then fade out
			dashboard.closeModal()
		}
	})
})
