# Example JSON log output:

```json
[
	{
		"level": "info",
		"path": "fileuploader.config.toml",
		"time": "2019-07-01T09:57:50-05:00",
		"message": "Loading config file"
	},
	{
		"level": "debug",
		"trustedCidrs": [
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
			"fc00::/7",
			"127.0.0.0/8",
			"::1/128"
		],
		"time": "2019-07-01T09:57:50-05:00",
		"message": "Trusting reverse proxies"
	},
	{
		"level": "debug",
		"size": "10MB",
		"time": "2019-07-01T09:57:50-05:00",
		"message": "Using upload limit"
	},
	{
		"level": "info",
		"event": "startup",
		"address": "192.168.1.11:40030",
		"time": "2019-07-01T09:57:50-05:00",
		"message": "Server listening"
	},
	{
		"level": "debug",
		"event": "http_request",
		"status": 200,
		"duration": 0.325652,
		"client": "192.168.1.11",
		"method": "OPTIONS",
		"path": "/files",
		"time": "2019-07-01T09:58:13-05:00",
		"message": "Handled request"
	},
	{
		"level": "warn",
		"error": "Issuer \"192.168.1.11\" not configured",
		"time": "2019-07-01T09:58:13-05:00",
		"message": "Failed to process EXTJWT"
	},
	{
		"level": "debug",
		"event": "http_request",
		"status": 201,
		"duration": 12.6806,
		"client": "192.168.1.11",
		"method": "POST",
		"path": "/files",
		"time": "2019-07-01T09:58:13-05:00",
		"message": "Handled request"
	},
	{
		"level": "debug",
		"id": "a32b4769c19af5cfdb95994df1fea80d",
		"ip": "192.168.1.11",
		"time": "2019-07-01T09:58:13-05:00",
		"message": "Recording uploader IP"
	},
	{
		"level": "info",
		"event": "post_create",
		"id": "a32b4769c19af5cfdb95994df1fea80d",
		"size": 779985,
		"offset": 0,
		"metadata": {
			"RemoteIP": "192.168.1.11",
			"extjwt": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2NvdW50IjoiIiwiY2hhbm5lbCI6IiIsImV4cCI6MTU2MTk5MzE1MywiaXNzIjoiMTkyLjE2OC4xLjExIiwiam9pbmVkIjpmYWxzZSwibW9kZXMiOltdLCJuZXRfbW9kZXMiOltdLCJuaWNrIjoiZmlyZWZveCIsInRpbWVfam9pbmVkIjowfQ.bE574RuQoZmZl9PjCuPOrRM6_hw32XkvUBFv8z0oFww",
			"filename": "Screenshot from 2019-06-10 05-54-18.png",
			"filetype": "image/png",
			"name": "Screenshot from 2019-06-10 05-54-18.png",
			"relativePath": "null",
			"type": "image/png"
		},
		"isPartial": false,
		"partialUploads": [],
		"time": "2019-07-01T09:58:13-05:00",
		"message": "Tusd post-create"
	},
	{
		"level": "debug",
		"event": "http_request",
		"status": 200,
		"duration": 0.011666,
		"client": "192.168.1.11",
		"method": "OPTIONS",
		"path": "/files/a32b4769c19af5cfdb95994df1fea80d",
		"time": "2019-07-01T09:58:13-05:00",
		"message": "Handled request"
	},
	{
		"level": "debug",
		"event": "http_request",
		"status": 204,
		"duration": 0.783014,
		"client": "192.168.1.11",
		"method": "PATCH",
		"path": "/files/a32b4769c19af5cfdb95994df1fea80d",
		"time": "2019-07-01T09:58:13-05:00",
		"message": "Handled request"
	},
	{
		"level": "info",
		"event": "post_receive",
		"id": "a32b4769c19af5cfdb95994df1fea80d",
		"size": 779985,
		"offset": 524288,
		"metadata": {
			"RemoteIP": "192.168.1.11",
			"extjwt": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2NvdW50IjoiIiwiY2hhbm5lbCI6IiIsImV4cCI6MTU2MTk5MzE1MywiaXNzIjoiMTkyLjE2OC4xLjExIiwiam9pbmVkIjpmYWxzZSwibW9kZXMiOltdLCJuZXRfbW9kZXMiOltdLCJuaWNrIjoiZmlyZWZveCIsInRpbWVfam9pbmVkIjowfQ.bE574RuQoZmZl9PjCuPOrRM6_hw32XkvUBFv8z0oFww",
			"filename": "Screenshot from 2019-06-10 05-54-18.png",
			"filetype": "image/png",
			"name": "Screenshot from 2019-06-10 05-54-18.png",
			"relativePath": "null",
			"type": "image/png"
		},
		"isPartial": false,
		"partialUploads": [],
		"time": "2019-07-01T09:58:13-05:00",
		"message": "Tusd post-receive"
	},
	{
		"level": "debug",
		"event": "upload_finished",
		"id": "a32b4769c19af5cfdb95994df1fea80d",
		"time": "2019-07-01T09:58:13-05:00",
		"message": "Finishing upload"
	},
	{
		"level": "debug",
		"event": "http_request",
		"status": 204,
		"duration": 13.289358,
		"client": "192.168.1.11",
		"method": "PATCH",
		"path": "/files/a32b4769c19af5cfdb95994df1fea80d",
		"time": "2019-07-01T09:58:13-05:00",
		"message": "Handled request"
	},
	{
		"level": "info",
		"event": "post_receive",
		"id": "a32b4769c19af5cfdb95994df1fea80d",
		"size": 779985,
		"offset": 779985,
		"metadata": {
			"RemoteIP": "192.168.1.11",
			"extjwt": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2NvdW50IjoiIiwiY2hhbm5lbCI6IiIsImV4cCI6MTU2MTk5MzE1MywiaXNzIjoiMTkyLjE2OC4xLjExIiwiam9pbmVkIjpmYWxzZSwibW9kZXMiOltdLCJuZXRfbW9kZXMiOltdLCJuaWNrIjoiZmlyZWZveCIsInRpbWVfam9pbmVkIjowfQ.bE574RuQoZmZl9PjCuPOrRM6_hw32XkvUBFv8z0oFww",
			"filename": "Screenshot from 2019-06-10 05-54-18.png",
			"filetype": "image/png",
			"name": "Screenshot from 2019-06-10 05-54-18.png",
			"relativePath": "null",
			"type": "image/png"
		},
		"isPartial": false,
		"partialUploads": [],
		"time": "2019-07-01T09:58:13-05:00",
		"message": "Tusd post-receive"
	},
	{
		"level": "info",
		"event": "post_finish",
		"id": "a32b4769c19af5cfdb95994df1fea80d",
		"size": 779985,
		"offset": 779985,
		"metadata": {
			"RemoteIP": "192.168.1.11",
			"extjwt": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2NvdW50IjoiIiwiY2hhbm5lbCI6IiIsImV4cCI6MTU2MTk5MzE1MywiaXNzIjoiMTkyLjE2OC4xLjExIiwiam9pbmVkIjpmYWxzZSwibW9kZXMiOltdLCJuZXRfbW9kZXMiOltdLCJuaWNrIjoiZmlyZWZveCIsInRpbWVfam9pbmVkIjowfQ.bE574RuQoZmZl9PjCuPOrRM6_hw32XkvUBFv8z0oFww",
			"filename": "Screenshot from 2019-06-10 05-54-18.png",
			"filetype": "image/png",
			"name": "Screenshot from 2019-06-10 05-54-18.png",
			"relativePath": "null",
			"type": "image/png"
		},
		"isPartial": false,
		"partialUploads": [],
		"time": "2019-07-01T09:58:13-05:00",
		"message": "Tusd post-finish"
	},
	{
		"level": "debug",
		"event": "http_request",
		"status": 200,
		"duration": 0.611154,
		"client": "192.168.1.11",
		"method": "GET",
		"path": "/files/a32b4769c19af5cfdb95994df1fea80d",
		"time": "2019-07-01T09:59:49-05:00",
		"message": "Handled request"
	},
	{
		"level": "warn",
		"event": "http_request",
		"status": 404,
		"duration": 0.007189,
		"client": "192.168.1.11",
		"method": "GET",
		"path": "/favicon.ico",
		"time": "2019-07-01T09:59:49-05:00",
		"message": "Handled request"
	},
	{
		"level": "debug",
		"event": "gc_tick",
		"time": "2019-07-01T10:02:50-05:00",
		"message": "Filestore GC tick"
	},
	{
		"level": "info",
		"event": "expired",
		"id": "e2d5b5c8704b720882fe309c4ff770f6",
		"time": "2019-07-01T10:02:50-05:00",
		"message": "Terminated upload id"
	}
]
```
