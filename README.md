# File sharing for [Kiwi IRC](https://kiwiirc.com)

* Upload files from you computer / device
* Take a webcam photo or video
* Paste files / images directly into Kiwi IRC
* Auto delete files after a time period

This plugin includes a file uploading server that will store any user uploaded files on the
server and then offer them as file downloads with a unique URL. The option to delete files
after a set time period discourages users from using the server as a permanent file store.

#### Dependencies
* yarn (https://yarnpkg.com/ - for the kiwiirc plugin UI)
* dep (https://github.com/golang/dep - for the server)

#### Running the file upload server

The file upload web server stores the files on the server and serves them to kiwi users.
```console
$ dep ensure
$ go run *.go
```

To build the server for production:
```console
$ go build -o fileuploader *.go
```

#### Building the Kiwi IRC plugin

The kiwi plugin is the javascript file that you link to in your kiwiirc configuration. It is the front end that provides the upload UI.

* `$ yarn start` will start a webpack development server that hot-reloads the plugin as you develop it. Use the URL `http://localhost:9000/main.js` as the plugin URL in your kiwiirc configuration.
* `$ yarn build` will build the final plugin that you can use in production. It will be built into dist/main.js.

##### Loading the plugin into kiwiirc
Add the plugin javascript file to your kiwiirc `config.json` and configure the settings:

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
		"note": "Add an optional note to the upload dialog"
	}
}
```

## Database configuration
File uploads are logged into a database. Currently the supported databases are sqlite and mysql.

* `DATABASE_TYPE` can either by `sqlite` or `mysql`. The default is `sqlite`.
* `DATABASE_PATH` is the path to your database file for sqlite. For mysql it is a DSN in the format `user:password@tcp(127.0.0.1:3306)/database`.

## License

[ Licensed under the Apache License, Version 2.0](LICENSE).
