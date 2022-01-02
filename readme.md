ytcast
======

cast YouTube videos to your smart TV from command-line.

this program does roughly the same thing as the "Play on TV" button that appears
on the player bar when you visit youtube.com with Chrome or when you use the
YouTube smartphone app:

![Play on TV button][0]

([the feature is also described here][1]).

I don't use Chrome as my daily driver because of *reasons* and I tend to use my
smartphone the least as possible when I'm at home... but still I want the "Play
on TV" functionality to watch videos on the big television screen without having
to search them with the remote! this is why I wrote this tool. also my computing
workflow is "command-line centric" and `ytcast` fits well in my toolbox.

[0]: play-on-tv.png
[1]: https://support.google.com/youtube/answer/7640706

usage
-----

https://user-images.githubusercontent.com/23704923/147848611-0d20563e-f656-487a-9774-9eb6feca1f58.mp4

([video demo on YouTube if above doesn't play][2]).

- the computer running `ytcast` and the target device need to be on the same
  network.
- the target device should have the YouTube app already installed.
- it also helps if the target device is already turned ON. `ytcast` supports
  Wake-on-Lan, but it's still WIP and doesn't work very well (see TODO).

run `ytcast -h` for the full usage, here I'll show the basic options.

the `-n` (name) option selects the target device matching by name or hostname
(ip):

    $ ytcast -n fire https://www.youtube.com/watch?v=dQw4w9WgXcQ

to see the already discovered (cached) devices, run `ytcast` without any
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

a `go` compiler is required for building, `make` is also nice to have.

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

to uninstall run `make uninstall` (with the same `PREFIX` used for `install`).

how it works
------------

I've always been curious to know how my phone can find my TV on my home network
and instruct it to start the YouTube app and play a video right away without
basically any manual pairing.

I did some research and found about this nice little protocol called [DIAL
(DIscovery And Launch)][6] developed by Netflix and Google which does the
initial part i.e. allows second-screen devices (phone, laptop, etc..) to
discover and launch apps on first-screen devices (TV, set-top, blu-ray, etc..).
there is a 40 pages [specification][7] and a [reference implementation][8] for
this protocol.

the discovery part of DIAL is actually performed using another protocol, [SSDP
(Simple Service Discovery Protocol)][9], which in turn is part of [UPnP][10].

all this is not enough to play videos. once the YouTube TV app is started by
DIAL, we need some other way to "tell" the app which video we want to play
(actually DIAL allows to pass parameters to an app you want to launch, but this
mechanism is not used by the YouTube TV app anymore).

after a little more research, I found about the YouTube Lounge api which is used
by Chrome and the YouTube smartphone app to remotely control the YouTube TV app.
it allows to start playing videos, pause, unpause, skip, add videos to the queue
and more. the api is **not documented** and understanding how it works it's not
that easy and fun. luckily lots of people have already reverse engineered the
thing (see THANKS) so all I had to do was taking the bits I needed to build
`ytcast`.

the bridge between DIAL and YouTube Lounge api is this `screenId` which as you
can imagine is an identifier for your "screen" (TV app). DIAL allows to get
information about the current "state" of an app on a particular device.  some
fields of this state are required by DIAL, other fields are app specific (called
additional data). `screenId` is a YouTube specific field that can be used to get
a token for the YouTube Lounge api: with that token we can control the TV app
via api calls.

putting all together, what `ytcast` does is:

1. search DIAL enabled devices on the local network (SSDP)
2. get the state of the YouTube TV app on the target device (DIAL)
3. if the app it's stopped, start it (DIAL)
4. get the `screenId` of the app (DIAL)
5. get an api token for that `screenId` (Lounge)
6. call the api's "play video endpoint" passing the token and the urls of the
   videos to play (Lounge)

(there is a "devices cache" involved so `ytcast` won't necessarily do all these
steps every time, also if the target device is turned off, `ytcast` tries to
wake it up with [Wake-on-Lan][11]).

as you maybe have already guessed, all this **can break at any time!** the
weakest point is the YouTube Lounge api since it's **not documented** and
`ytcast` depends heavily on it.

**`ytcast` may not work at all on your setup!** I use and test `ytcast` with 2
devices:

- an Amazon Fire TV Stick
- a LG Smart TV running WebOS

that's all I have. `ytcast` works great with both these devices but I don't know
if it will work well on different setups (it should, but I don't know for sure).
if it doesn't work on your setup please [open an issue][12] describing your
setup and attach a `-verbose` log so we can investigate what's wrong and
hopefully fix it.

also **chromecast**. I don't own a chromecast and `ytcast` probably won't work
with chromecast because it doesn't use the DIAL protocol anymore (at least
that's what I've read somewhere). `ytcast` (should) work any with DIAL enabled
devices that supports the YouTube TV app.

[6]: http://www.dial-multiscreen.org
[7]: http://www.dial-multiscreen.org/dial-protocol-specification/DIAL-2ndScreenProtocol-2.2.1.pdf
[8]: https://github.com/Netflix/dial-reference
[9]: https://en.wikipedia.org/wiki/Simple_Service_Discovery_Protocol
[10]: https://en.wikipedia.org/wiki/Universal_Plug_and_Play
[11]: https://en.wikipedia.org/wiki/Wake-on-LAN
[12]: https://github.com/MarcoLucidi01/ytcast/issues

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
