libman
---------

libman is a CLI client for controlling your Spotify playback, it also lets you manage your playlists.

Features
----------

You can do pretty much anything, some features are not implemented but soon will be.

implemented:

-	Control playback, volume and device.
-	Create/ edit your playlists.
-	Search for songs, playlists, artists and albums.

Not yet implemented:

-	Deleting/ unfollowing a playlist.
-	View/ edit/ listen to your "favourites" folder.

Maybe will get implemented:

-	Playing/ following a podcast.

Installation
---------

You need a working go installation, preferably the latest version and have go modules enabled.

	go get -u -v github.com/insomnimus/libman/...

Go should install libman for you, if it doesn't, try these steps:

-	`git clone --recurse  https://github.com/insomnimus/libman`
-	`cd libman`
-	`go install`

After the installation, you probably should configure libman.

Configuration
---------

-	First things first, you need to register a new application at [Spotify](https://developer.spotify.com/my-applications/).
-	Set `http://localhost:8080/callback` as the callback URI, this is important.
-	Now set the `SPOTIFY_ID` and `SPOTIFY_SECRET` environment variable to what you got in step 1.

If you don't want to use environment variables, run `libman config` for the location of the config file and put them there`

You're ready, start libman by calling it bare:

	libman
