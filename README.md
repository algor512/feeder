This is the tool I use to read my personal news feed.

It uses the Go's standard http and https servers (see `-https` argument) to serve the feed pages. The
backend (see `utils/provider.go`) periodically updates the feed and the pages. In fact, it works like a
static website with all pages stored in RAM, so it provides very quick response times.

I made this tool for personal use. However, feel free to fork it and use it for your purposes. I
am also open to any criticism.
