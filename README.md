# Development dependencies

* yarn
* [devrun]
* [dep]

# Instructions

```console
$ dep ensure
$ devrun watch "go build && ./fileuploader"
$ cd fileuploader-kiwiirc-plugin && yarn serve
```

Add the plugin to your kiwiirc `config.json` and configure the settings:

```json
{
	"plugins": [
		{
			"name": "fileuploader",
			"url": "http://localhost:9000/main.js"
		}
	],
	"fileuploader": {
		"server": "http://localhost:8088/files",
		"maxFileSize": 10485760
	}
}
```

[devrun]: https://github.com/kdar/devrun
[dep]: https://github.com/golang/dep
