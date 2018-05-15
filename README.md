# File sharing for [Kiwi IRC](https://kiwiirc.com)

* Upload files from you computer / device
* Take a webcam photo or video
* Paste files / images directly into Kiwi IRC
* Auto delete files after a time period

This plugin includes a file uploading server that will store any user uploaded files on the
server and then offer them as file downloads with a unique URL. The option to delete files
after a set time period discourages users from using the server as a permanent file store.

## Development

**Dependencies**
* yarn
* devrun (optional, https://github.com/kdar/devrun)
* dep (optional, https://github.com/golang/dep)

**Running with live-reload of the server and client plugin**
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
		"maxFileSize": 10485760,
		"note": "Add am optional note to the upload dialog"
	}
}
```

## Database configuration
File uploads are logged into a database. Currently the supported databases are sqlite and mysql.

* `DATABASE_TYPE` can either by `sqlite` or `mysql`. The default is `sqlite`.
* `DATABASE_PATH` is the path to your database file for sqlite. For mysql it is a DSN in the format `user:password@tcp(127.0.0.1:3306)/database`.

## License

[ Licensed under the Apache License, Version 2.0](LICENSE).
