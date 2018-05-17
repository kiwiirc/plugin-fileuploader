import { Core as Uppy, Dashboard, Tus, Webcam } from 'uppy'
import 'uppy/dist/uppy.min.css'

const KiB = 2 ** 10
const MiB = 2 ** 20

function getValidUploadTarget() {
	const buffer = kiwi.state.getActiveBuffer()
	const isValidTarget = buffer && (buffer.isChannel() || buffer.isQuery())
	if (!isValidTarget) {
		throw new Error('Files can only be shared in channels or queries.')
	}
	return buffer
}

kiwi.plugin('fileuploader', function(kiwi, log) {
	const settings = kiwi.state.setting('fileuploader')

	// add button to input bar
	const uploadFileButton = document.createElement('i')
	uploadFileButton.className = 'upload-file-button fa fa-upload'

	kiwi.addUi('input', uploadFileButton)

	const uppy = Uppy({
		autoProceed: false,
		onBeforeFileAdded: (currentFile, files) => {
			// throws if invalid, canceling the file add
			getValidUploadTarget()
		},
		restrictions: {
			maxFileSize: settings.maxFileSize || 10 * MiB,
		},
	})
		.use(Dashboard, {
			trigger: uploadFileButton,
			proudlyDisplayPoweredByUppy: false,
			note: settings.note,
		})
		.use(Webcam, { target: Dashboard })
		.use(Tus, {
			endpoint: settings.server || '/files',
			chunkSize: 512 * KiB,
		})
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

	uppy.on('file-added', file => {
		file.kiwiFileUploaderTargetBuffer = getValidUploadTarget()
	})

	uppy.on('upload-success', (file, resp, uploadURL) => {
		// append filename to uploadURL
		uploadURL = `${uploadURL}/${encodeURIComponent(file.meta.name)}`
		uppy.setFileState(file.id, { uploadURL })
		file = uppy.getFile(file.id)

		// emit a global kiwi event
		kiwi.emit('fileuploader.uploaded', { url: uploadURL, file })

		// send a message with the url of each successful upload
		const buffer = file.kiwiFileUploaderTargetBuffer
		buffer.say(`Uploaded file: ${uploadURL}`)
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
