# amahi-anywhere-fs
[Amahi Anywhere](https://www.amahi.org/apps/amahi-anywhere) streaming file server

Amahi Anywhere is a protocol and a suite of apps, like the [iOS](https://www.amahi.org/ios) and [Android](https://www.amahi.org/android) clients to access, view and stream files in your Amahi server.

To compile your own version of this server you will need to have a file with API configuration secrets in `src/fs/secrets.go` containing:

```golang
package main

const TMDB_API_KEY = "abc"
const TVRAGE_API_KEY = "def"
const TVDB_API_KEY = "ghi"

const PFE_HOST = "yourserver"
const PFE_PORT = "1234567"
const SECRET_TOKEN = "xyz"
```
