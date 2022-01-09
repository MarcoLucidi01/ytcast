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
workflow is "command-line centric" and `ytcast` fits well in my toolbox (see
[other tools][2]).

[0]: play-on-tv.png
[1]: https://support.google.com/youtube/answer/7640706
[2]: #other-tools

usage
-----

https://user-images.githubusercontent.com/23704923/147848611-0d20563e-f656-487a-9774-9eb6feca1f58.mp4

([video demo on YouTube if above doesn't play][3]).

- the computer running `ytcast` and the target device need to be on the **same
  network**.
- the target device should have the YouTube on TV app already installed.

run `ytcast -h` for the full usage, here I'll show the basic options.

the `-n` (name) option selects the target device matching by name, hostname
(ip), or unique service name:

    $ ytcast -n fire https://www.youtube.com/watch?v=dQw4w9WgXcQ

to see the already discovered (cached) devices, run `ytcast` without any option:

    $ ytcast
    28bc7426 192.168.1.35    "FireTVStick di Marco"         cached lastused
    d0881fbe 192.168.1.227   "[LG] webOS TV UM7100PLB"      cached
    ytcast: no device selected

to update the devices cache use the `-s` (search) option (it's implicit when the
cache is empty or when `-n` doesn't match anything):

    $ ytcast -s
    28bc7426 192.168.1.35    "FireTVStick di Marco"         lastused
    d0881fbe 192.168.1.227   "[LG] webOS TV UM7100PLB"      cached

if your target device doesn't show up, you can try increasing the search timeout
with the `-t` (timeout) option to give the device more time to respond to the
query (default is 3 seconds):

    $ ytcast -s -t 5s
    28bc7426 192.168.1.35    "FireTVStick di Marco"         lastused
    d0881fbe 192.168.1.227   "[LG] webOS TV UM7100PLB"      cached

(remember that the computer and the target device must be on the same network).

to cast to the last used device use the `-l` option:

    $ ytcast -l https://www.youtube.com/watch?v=dQw4w9WgXcQ

`ytcast` can also read video urls (or ids) from `stdin` one per line:

    $ ytcast -n lg < watchlist

this makes it easy to combine `ytcast` with other tools like [`ytfzf`][5] or my
`ytfzf` clone [`ytsearch`][6].

to see what's going on under the hood use the `-verbose` option:

    $ ytsearch fireplace 10 hours | ytcast -n lg -verbose
    21:13:08 ytcast.go:82: ytcast v0.1.0-6-g8e6daeb
    21:13:08 ytcast.go:168: mkdir -p /home/marco/.cache/ytcast
    21:13:08 ytcast.go:177: loading cache /home/marco/.cache/ytcast/ytcast.json
    21:13:08 ytcast.go:319: reading videos from stdin
    21:13:15 dial.go:153: GET http://192.168.1.227:1754/
    21:13:15 dial.go:153: GET http://192.168.1.227:36866/apps/YouTube
    21:13:15 ytcast.go:293: "YouTube" is stopped on "[LG] webOS TV UM7100PLB"
    21:13:15 ytcast.go:306: launching "YouTube" on "[LG] webOS TV UM7100PLB"
    21:13:15 dial.go:153: POST http://192.168.1.227:36866/apps/YouTube
    21:13:18 dial.go:153: GET http://192.168.1.227:36866/apps/YouTube
    21:13:18 ytcast.go:293: "YouTube" is running on "[LG] webOS TV UM7100PLB"
    21:13:18 ytcast.go:358: requesting YouTube Lounge to play [cdKop6aixVE] on "[LG] webOS TV UM7100PLB"
    21:13:18 remote.go:233: POST https://www.youtube.com/api/lounge/bc/bind
    21:13:18 remote.go:233: POST https://www.youtube.com/api/lounge/bc/bind
    21:13:18 ytcast.go:197: saving cache /home/marco/.cache/ytcast/ytcast.json

(please run with `-verbose` and **attach the log** when reporting an [issue][7]).

[3]: https://www.youtube.com/watch?v=07aWOpi8DVk
[5]: https://github.com/pystardust/ytfzf
[6]: https://github.com/MarcoLucidi01/bin/blob/master/ytsearch
[7]: https://github.com/MarcoLucidi01/ytcast/issues

build and install
-----------------

a `go` compiler is required for building, `make` is also nice to have (binary
releases will come someday).

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
and instruct it to start the YouTube on TV app and play a video right away
without basically any manual pairing.

I did some research and found about this nice little protocol called [DIAL
(DIscovery And Launch)][8] developed by Netflix and Google which does the
initial part i.e. allows second-screen devices (phone, laptop, etc..) to
discover and launch apps on first-screen devices (TV, set-top, blu-ray, etc..).
there is a 40 pages [specification][9] and a [reference implementation][10] for
this protocol.

the discovery part of DIAL is actually performed using another protocol, [SSDP
(Simple Service Discovery Protocol)][11], which in turn is part of [UPnP][12].

all this is not enough to play videos. once the YouTube on TV app is started by
DIAL, we need some other way to "tell" the app which video we want to play
(actually DIAL allows to pass parameters to an app you want to launch, but this
mechanism is not used by the YouTube on TV app anymore).

after a little more research, I found about the YouTube Lounge api which is used
by Chrome and the YouTube smartphone app to remotely control the YouTube on TV
app. it allows to start playing videos, pause, unpause, skip, add videos to the
queue and more. the api is **not documented** and understanding how it works
it's not an easy and fun job. luckily lots of people have already reverse
engineered the thing (see [THANKS][13]) so all I had to do was taking the bits I
needed to build `ytcast`.

the bridge between DIAL and YouTube Lounge api is the `screenId` which as you
can imagine is an identifier for your "screen" (TV app). DIAL allows to get
information about the current "state" of an app on a particular device. some
fields of this state are required by DIAL, other fields are app specific (called
additional data). `screenId` is a YouTube specific field that can be used to get
a token from the YouTube Lounge api: with that token we can control the YouTube
on TV app via api calls.

putting all together, what `ytcast` does is:

1. search DIAL enabled devices on the local network (SSDP)
2. get the state of the YouTube on TV app on the target device (DIAL)
3. if the app it's stopped, start it (DIAL)
4. get the `screenId` of the app (DIAL)
5. get a token for that `screenId` (Lounge)
6. call the api's "play video endpoint" passing the token and the video urls to
   play (Lounge)

(there is a "devices cache" involved so `ytcast` won't necessarily do all these
steps every time, also if the target device is turned off, `ytcast` tries to
wake it up with [Wake-on-Lan][14]).

as you maybe have already guessed, all this **can stop working at any time!**
the weakest point is the YouTube Lounge api since it's **not documented** and
`ytcast` depends heavily on it. moreover, **`ytcast` may not work at all on your
setup!** I use and test `ytcast` with 2 devices:

- Amazon Fire TV Stick
- LG Smart TV running WebOS

that's all I have. `ytcast` works great with both these devices but I don't know
if it will work well on setups different than mine (it should, but I don't know
for sure). if it doesn't work on your setup please [open an issue][15]
describing your setup and attach a `-verbose` log so we can investigate what's
wrong and hopefully fix it.

also **Chromecast**. I don't own a Chromecast and `ytcast` probably won't work
with Chromecast because it doesn't use the DIAL protocol anymore (at least
that's what I've read somewhere). `ytcast` (should) work with any DIAL enabled
device that supports the YouTube on TV app.

[8]: http://www.dial-multiscreen.org
[9]: http://www.dial-multiscreen.org/dial-protocol-specification/DIAL-2ndScreenProtocol-2.2.1.pdf
[10]: https://github.com/Netflix/dial-reference
[11]: https://en.wikipedia.org/wiki/Simple_Service_Discovery_Protocol
[12]: https://en.wikipedia.org/wiki/Universal_Plug_and_Play
[13]: #thanks
[14]: https://en.wikipedia.org/wiki/Wake-on-LAN
[15]: https://github.com/MarcoLucidi01/ytcast/issues

THANKS
------

I would like to thank all the people whose work has helped me tremendously in
building `ytcast`, especially the following projects/posts:

- https://0x41.cf/automation/2021/03/02/google-assistant-youtube-smart-tvs.html
- https://github.com/thedroidgeek/youtube-cast-automation-api
- https://github.com/mutantmonkey/youtube-remote
- https://bugs.xdavidhu.me/google/2021/04/05/i-built-a-tv-that-plays-all-of-your-private-youtube-videos
- https://github.com/aykevl/plaincast
- https://github.com/ur1katz/casttube

TODO
----

- [ ] add flag to add videos to playing queue (`-a`).
- [ ] add flag to disconnect from device (`-d`)? not a priority.
- [ ] allow to play videos from specific timestamp? might be useful.
- [ ] playlist urls don't work!

other tools
-----------

as I said earlier, my computing environment is very command-line centric and I'd
like to showcase the other tools I use to enjoy a "no frills" YouTube experience
from the terminal!

- [`youtube-dl`][16] (actually [`yt-dlp`][17] these days) doesn't need
  introduction, it's an awesome tool and it's well integrated with [`mpv`][18]
  so I can watch videos with my favorite player without having my laptop fan
  spin like an airplane engine thanks to this `mpv` config:

      ytdl-format=bestvideo[height<=?1080][vcodec!=?vp9]+bestaudio/best

- [`ytsearch`][19] is my clone of the initial version of [`ytfzf`][20]. it
  allows to search and select video urls from the command-line using the
  wonderful [`fzf`][21] (fun fact: it's implemented basically as a single big
  pipe ahah). you have already seen it in action in `ytcast` examples, but it
  works great with `mpv` too:

      $ ytsearch matrix 4 | xargs mpv
      $ ytsearch 9 symphony | xargs mpv --no-video

- [`ytxrss`][22] allows to extract the rss feed url of a YouTube channel
  starting from a video or channel url. I use rss feeds ([`newsboat`][23]) to
  keep up-to-date with *things* and I'm really glad YouTube still supports them
  for channel uploads. if I'm interested in a channel's future uploads, what I
  usually do is:

      $ ytxrss https://www.youtube.com/user/Computerphile >> ~/.newsboat/urls

[16]: https://github.com/ytdl-org/youtube-dl
[17]: https://github.com/yt-dlp/yt-dlp
[18]: https://github.com/mpv-player/mpv
[19]: https://github.com/MarcoLucidi01/bin/blob/master/ytsearch
[20]: https://github.com/pystardust/ytfzf
[21]: https://github.com/junegunn/fzf
[22]: https://github.com/MarcoLucidi01/bin/blob/master/ytxrss
[23]: https://github.com/newsboat/newsboat

---

see [license file][50] for copyright and license details.

[50]: license
