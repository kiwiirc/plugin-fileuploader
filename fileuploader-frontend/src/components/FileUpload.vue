<template>
	<div class="file-upload">
		<div
			class="drop-zone"
			v-on:dragenter="dragenter"
			v-on:dragover="dragover"
			v-on:dragleave="dragleave"
			v-on:drop="drop"
			v-bind:class="{ 'drop-hover': isDropHovered }"
		>
			Drop a file here
		</div>
	</div>
</template>

<script>
import tus from 'tus-js-client'

function mustUnwrapSingle(list) {
	if (list.length !== 1) {
		throw new TypeError(`expected length 1, got ${list.length}`)
	}
	return list[0]
}

// const tusOptions = {
// 	endpoint: 'http://127.0.0.1:8088/files',
// 	onError : function (error) {
// 		debugger
// 		if (error.originalRequest) {
// 			if (window.confirm("Failed because: " + error + "\nDo you want to retry?")) {
// 				upload.start();
// 				uploadIsRunning = true;
// 				return;
// 			}
// 		} else {
// 			window.alert("Failed because: " + error);
// 		}

// 		// reset();
// 	},
// 	onProgress: function (bytesUploaded, bytesTotal) {
// 		var percentage = (bytesUploaded / bytesTotal * 100).toFixed(2);
// 		// progressBar.style.width = percentage + "%";
// 		console.log(bytesUploaded, bytesTotal, percentage + "%");
// 	},
// 	onSuccess: function () {
// 		console.log("success", arguments)
// 		// debugger
// 		// var anchor = document.createElement("a");
// 		// anchor.textContent = "Download " + upload.file.name + " (" + upload.file.size + " bytes)";
// 		// anchor.href = upload.url;
// 		// anchor.className = "btn btn-success";
// 		// uploadList.appendChild(anchor);

// 		// reset();
// 	}
// }

export default {
	name: 'FileUpload',
	props: {},
	data: () => ({
		isDropHovered: false
	}),
	methods: {
		dragenter(event) {
			this.isDropHovered = true
			event.preventDefault()
		},
		dragover(event) {
			event.preventDefault()
		},
		dragleave() {
			this.isDropHovered = false
		},
		drop(event) {
			event.preventDefault()
			this.isDropHovered = false
			const { files } = event.dataTransfer
			const file = mustUnwrapSingle(files)
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
		}
	}
}
</script>

<style scoped>
.drop-zone {
	width: 50vmin;
	height: 50vmin;
	border: 1px solid #d6d3d3;
}
.drop-hover {
	border: 1px dashed red;
}
</style>
