/*
Command mimegen generates a Go MimeTypeID registry from mime-db JSON.

Usage:

	mimegen [-in db.json|-] [-url https://...] [-pkg mimeid] [-o mimeids_gen.go]

Examples:

	# from a local file
	mimegen -in db.json -pkg api -o api/mimeids_gen.go

	# from the network (pin a ref!)
	mimegen -url https://raw.githubusercontent.com/jshttp/mime-db/v1.54.0/db.json

	# from stdin
	curl -sL .../db.json | mimegen -

Input format:

	# example JSON database content (db.json):
	{
		"application/dotsig-sig+json": {
			"source": "iana",
			"charset": "UTF-8",
			"compressible": true,
			"extensions": []
		}
	}

Note that only the `source` field is *always* included.
*/
package main
