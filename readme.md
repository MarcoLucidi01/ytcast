ytcast
======

cast YouTube videos to your smart TV from command-line.

this program does roughly the same thing as the "Play on TV" button that appears
on the player bar when you visit youtube.com with Chrome or when you use the
YouTube smartphone app:

![Play on TV button][0]

(the [feature is also described here][1]).

I don't use Chrome as my daily driver because of *reasons* and I tend to use my
smartphone the least as possible when I'm at home... but still I want the "Play
on TV" functionality to watch videos on the big television screen without having
to search them with the remote! this is why I wrote this tool. also my computing
workflow is "command-line centric" and `ytcast` fits well in my toolbox.

[0]: play-on-tv.png
[1]: https://support.google.com/youtube/answer/7640706

usage
-----

([video demo on YouTube][2]).

the computer that runs `ytcast` and the target device need to be on the same
network.

    $ ytcast -n fire https://www.youtube.com/watch?v=dQw4w9WgXcQ

the `-n` (name) option selects the target device matching by name or hostname
(ip). to see the already discovered (cached) devices, run `ytcast` without any
option:

    $ ytcast
    "FireTVStick di Marco"         192.168.1.23    cached lastused
    "[LG] webOS TV UM7100PLB"      192.168.1.227   cached
    ytcast: no device selected

to update the devices cache use the `-s` (search) option (it's implicit when the
cache is empty):

    $ ytcast -s
    "FireTVStick di Marco"         192.168.1.23    lastused
    "[LG] webOS TV UM7100PLB"      192.168.1.227   cached

if your target device doesn't show up, you can try increasing the search
timeout with the `-t` (timeout) option to give the device more time to respond
to the query (default is 3 seconds):

    $ ytcast -s -t 5s
    "FireTVStick di Marco"         192.168.1.23    lastused
    "[LG] webOS TV UM7100PLB"      192.168.1.227   cached

(remember that the computer and the target device must be on the same network).

to cast to the last used device use the `-l` option:

    $ ytcast -l https://www.youtube.com/watch?v=dQw4w9WgXcQ

`ytcast` can also read video urls (or ids) from `stdin` one per line:

    $ ytcast -n lg < watchlist

this makes it easy to combine `ytcast` with other tools like [`ytfzf`][3] or my
`ytfzf` clone [`ytsearch`][4].

to see what's going on under the hood use the `-verbose` option:

    $ ytsearch fireplace 10 hours | ytcast -n lg -verbose
    21:13:08.724933 ytcast.go:82: ytcast v0.1.0-6-g8e6daeb
    21:13:08.725031 ytcast.go:168: mkdir -p /home/marco/.cache/ytcast
    21:13:08.725061 ytcast.go:177: loading cache /home/marco/.cache/ytcast/ytcast.json
    21:13:08.725501 ytcast.go:319: reading videos from stdin
    21:13:15.585240 dial.go:153: GET http://192.168.1.227:1754/
    21:13:15.752052 dial.go:153: GET http://192.168.1.227:36866/apps/YouTube
    21:13:15.951936 ytcast.go:293: "YouTube" is stopped on "[LG] webOS TV UM7100PLB"
    21:13:15.951969 ytcast.go:306: launching "YouTube" on "[LG] webOS TV UM7100PLB"
    21:13:15.951996 dial.go:153: POST http://192.168.1.227:36866/apps/YouTube
    21:13:18.258981 dial.go:153: GET http://192.168.1.227:36866/apps/YouTube
    21:13:18.276945 ytcast.go:293: "YouTube" is running on "[LG] webOS TV UM7100PLB"
    21:13:18.277112 ytcast.go:358: requesting YouTube Lounge to play [cdKop6aixVE] on "[LG] webOS TV UM7100PLB"
    21:13:18.277145 remote.go:233: POST https://www.youtube.com/api/lounge/bc/bind
    21:13:18.717665 remote.go:233: POST https://www.youtube.com/api/lounge/bc/bind
    21:13:18.800910 ytcast.go:197: saving cache /home/marco/.cache/ytcast/ytcast.json

(please run with `-verbose` and **attach the log** when reporting an [issue][5]).

[2]: https://www.youtube.com/watch?v=07aWOpi8DVk
[3]: https://github.com/pystardust/ytfzf
[4]: https://github.com/MarcoLucidi01/bin/blob/master/ytsearch
[5]: https://github.com/MarcoLucidi01/ytcast/issues

build and install
-----------------

    $ git clone https://github.com/MarcoLucidi01/ytcast.git
    ...
    $ cd ytcast
    $ make install
    ...

`make install` installs in `/usr/local/bin` by default, you can change `PREFIX`
if you want, for example I like to keep my binaries inside `$HOME/bin` so I
usually install with:

    $ make install PREFIX=$HOME
    ...
    go build -o ytcast -ldflags="-X main.progVersion=v0.1.0-7-ge9c96d3"
    mkdir -p /home/marco/bin
    install -m 755 ytcast /home/marco/bin

how it works
------------

TODO

TODO can break at any time!

TODO things I can test

TODO things I cannot test (chromecast)

THANKS
------

TODO

TODO
----

- [ ] add flag to add videos to playing queue (`-a`).
- [ ] add flag to clear devices cache (`-c`). can be used with other flags?
- [ ] add flag to disconnect from device (`-d`)? not a priority.
- [ ] allow to play videos from specific timestamp? might be useful.
- [ ] allow `-s` to be used with `-l` and `-n` i.e. search and play.
- [ ] fix ping and wakeup issue (see TODO in ytcast.go).
- [ ] report error when more than one device matches `-n`.

other tools
-----------

TODO show case other command-line youtube tools I use

see license file for copyright and license details.
