# libman

## Authors Note

Since this project is one of my earlier go apps, the code is very disorganized and messy. 
Instead of fixing the code, i wanted to rewrite libman in rust, please check [this link](https://github.com/insomnimus/libman) out. 

On another note, I use libman daily, so it's ready to use but i'm not happy with the code (app works great).

libman is a CLI client for controlling your Spotify playback, it also lets you manage your playlists.

## Features

You can do pretty much anything, some features are not implemented but soon will be.

implemented:

-	Control playback, volume and device.
-	Create/ edit your playlists.
-	Search for songs, playlists, artists and albums.
-	View/ edit/ listen to your "favourites" folder.

Maybe will get implemented:

-	Playing/ following a podcast.

## Installation

Since libman v0.12.0, libman needs go 1.16 and above to be compiled.

`go install github.com/insomnimus/go-libman@latest`

If the above doesn't work, try cloning the repo first:

```sh
git clone https://github.com/insomnimus/go-libman
cd go-libman
git checkout dev
go install
```

After the installation, you probably should configure libman.

### Configuration

-	First things first, you need to register a new application at [Spotify](https://developer.spotify.com/my-applications/).
-	Set `http://localhost:8080/callback` as the callback URI, this is important.
-	Now set the `SPOTIFY_ID` and `SPOTIFY_SECRET` environment variable to what you got in step 1.

If you don't want to use environment variables, run `libman config` for the location of the config file and put them there

You're ready, start libman by calling it bare:

`go-libman`
