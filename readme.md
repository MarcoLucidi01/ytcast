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

https://user-images.githubusercontent.com/23704923/147848611-0d20563e-f656-487a-9774-9eb6feca1f58.mp4

([video demo on YouTube if above doesn't play][3]).

[0]: play-on-tv.png
[1]: https://support.google.com/youtube/answer/7640706
[2]: #other-tools
[3]: https://www.youtube.com/watch?v=07aWOpi8DVk

contents
--------

- [usage](#usage)
- [install](#install)
- [how it works](#how-it-works)
- [BUGS](#bugs)
- [workarounds](#workarounds)
- [THANKS](#thanks)
- [other tools](#other-tools)

usage
-----

- the computer running `ytcast` and the target device must be on the **same network**.
- the target device must support the **DIAL protocol** (see [how it works][14]).
- the target device must have the **YouTube on TV app already installed**.

run `ytcast -h` for the full usage, here I'll show the basic options.

the `-d` (device) option selects the target device matching by name, hostname
(ip), or unique service name:

    $ ytcast -d fire https://www.youtube.com/watch?v=dQw4w9WgXcQ

to see the already discovered (cached) devices use the `-l` (list) option:

    $ ytcast -l
    28bc7426 192.168.1.35    "FireTVStick di Marco"         cached lastused
    d0881fbe 192.168.1.227   "[LG] webOS TV UM7100PLB"      cached

to update the devices cache use the `-s` (search) option (it's implicit when the
cache is empty or when `-d` doesn't match anything in the cache):

    $ ytcast -s
    28bc7426 192.168.1.35    "FireTVStick di Marco"         lastused
    d0881fbe 192.168.1.227   "[LG] webOS TV UM7100PLB"      cached

if your target device doesn't show up, you can try increasing the search timeout
with the `-t` (timeout) option to give the device more time to respond to the
query:

    $ ytcast -s -t 10s
    28bc7426 192.168.1.35    "FireTVStick di Marco"         lastused
    d0881fbe 192.168.1.227   "[LG] webOS TV UM7100PLB"      cached

remember that the computer and the target device must be on the same network.
if it doesn't show up after several tries, you may consider using the `-pair`
option to skip the discovery process altogether. this adds some limitations
though, see [workarounds][15].

to cast to the last used device use the `-p` option:

    $ ytcast -p https://www.youtube.com/watch?v=dQw4w9WgXcQ

when no url is passed in the arguments, `ytcast` reads video urls (or ids) from
`stdin` one per line:

    $ ytcast -d lg < watchlist

this makes it easy to combine `ytcast` with other tools like [`ytfzf`][11] or my
`ytfzf` clone [`ytsearch`][12].

to see what's going on under the hood use the `-verbose` option:

    $ ytsearch fireplace 10 hours | ytcast -d lg -verbose
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

(please run with `-verbose` and **attach the log** when reporting an [issue][13]).

[11]: https://github.com/pystardust/ytfzf
[12]: https://github.com/MarcoLucidi01/bin/blob/master/ytsearch
[13]: https://github.com/MarcoLucidi01/ytcast/issues
[14]: #how-it-works
[15]: #workarounds

install
-------

you can get a pre-compiled binary from the [latest release][20] assets and copy
it somewhere in your `$PATH`.

here a quick and dirty one-liner script to do it fast on unix-like systems
(adjust `target` and `dir` to your needs, lookup available targets in the
[latest release][20] assets):

    (target="linux-amd64"; dir="$HOME/bin"; \
      wget -O - https://api.github.com/repos/MarcoLucidi01/ytcast/releases/latest \
        | jq -r --arg target "$target" '.assets[] | select(.name | match("checksums|"+$target)) | .browser_download_url' \
        | wget -i - \
       && sha256sum -c --ignore-missing ytcast-v*-checksums.txt \
       && tar -vxf ytcast-v*"$target.tar.gz" \
       && install -m 755 ytcast-v*"$target/ytcast" "$dir")

if you run Arch Linux (btw I don't) you can get [`ytcast-bin` from the AUR][21]
(many thanks to the maintainer)!

if your os or architecture are not available, or you want to get the latest
changes from `master`, you can compile from source. a `go` compiler and `make`
are required for building and installing:

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
    go build -trimpath -tags netgo,osusergo -ldflags="-w -s -X main.progVersion=v0.5.0-3-gd513b8e" -o ytcast
    mkdir -p /home/marco/bin
    install -m 755 ytcast /home/marco/bin

to uninstall run `make uninstall` (with the same `PREFIX` used for `install`).

[20]: https://github.com/MarcoLucidi01/ytcast/releases/latest
[21]: https://aur.archlinux.org/packages/ytcast-bin

how it works
------------

I've always been curious to know how my phone can find my TV on my home network
and instruct it to start the YouTube on TV app and play a video right away
without basically any manual pairing.

I did some research and found about this nice little protocol called [DIAL
(DIscovery And Launch)][30] developed by Netflix and Google which does the
initial part i.e. allows second-screen devices (phone, laptop, etc..) to
discover and launch apps on first-screen devices (TV, set-top, blu-ray, etc..).
there is a 40 pages [specification][31] and a [reference implementation][32] for
this protocol.

the discovery part of DIAL is actually performed using another protocol, [SSDP
(Simple Service Discovery Protocol)][33], which in turn is part of [UPnP][34].

all this is not enough to play videos. once the YouTube on TV app is started by
DIAL, we need some other way to "tell" the app which video we want to play
(actually DIAL allows to pass parameters to an app you want to launch, but this
mechanism is not used by the YouTube on TV app anymore).

after a little more research, I found about the YouTube Lounge api which is used
by Chrome and the YouTube smartphone app to remotely control the YouTube on TV
app. it allows to start playing videos, pause, unpause, skip, add videos to the
queue and more. the api is **not documented** and understanding how it works
it's not an easy and fun job. luckily lots of people have already reverse
engineered the thing (see [THANKS][35]) so all I had to do was taking the bits I
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
3. if the app is stopped, start it (DIAL)
4. get the `screenId` of the app (DIAL)
5. get a token for that `screenId` (Lounge)
6. call the api's "play video endpoint" passing the token and the video urls to
   play (Lounge)

(there is a "devices cache" involved so `ytcast` won't necessarily do all these
steps every time, also if the target device is turned off, `ytcast` tries to
wake it up with [Wake-on-Lan][36]).

as you may have already guessed, all this **can stop working at any time!** the
weakest point is the YouTube Lounge api since it's **not documented** and
`ytcast` depends heavily on it. moreover, **`ytcast` may not work at all on your
setup!** I use and test `ytcast` with 2 devices:

- Amazon Fire TV Stick
- LG Smart TV running WebOS

that's all I have. `ytcast` works great with both these devices but I don't know
if it will work well on setups different than mine (it should, but I don't know
for sure). if it doesn't work on your setup please [open an issue][37]
describing your setup and attach a `-verbose` log so we can investigate what's
wrong and hopefully fix it.

also **Chromecast**. I don't own a Chromecast and you'll probably need to use
the `-pair` option (see [workarounds][38]) to make `ytcast` work with Chromecast
because it [doesn't use the DIAL protocol anymore, but switched to mDNS for
discovery][39].

[30]: http://www.dial-multiscreen.org
[31]: http://www.dial-multiscreen.org/dial-protocol-specification/DIAL-2ndScreenProtocol-2.2.1.pdf
[32]: https://github.com/Netflix/dial-reference
[33]: https://en.wikipedia.org/wiki/Simple_Service_Discovery_Protocol
[34]: https://en.wikipedia.org/wiki/Universal_Plug_and_Play
[35]: #thanks
[36]: https://en.wikipedia.org/wiki/Wake-on-LAN
[37]: https://github.com/MarcoLucidi01/ytcast/issues
[38]: #workarounds
[39]: https://en.wikipedia.org/wiki/Chromecast#Device_discovery_protocols

BUGS
----

- sometimes the playing queue gets "messed up" i.e. some videos are added
  between others, some videos don't get added at all or even an old queue might
  be "reused" so you could see videos from an old session after the ones you
  requested to play. unfortunately, using (or misusing) an undocumented api may
  lead to these kinds of problems and I haven't bothered too much trying to fix
  them.

- the `-a` (add) option is slower because it does an api call for each video you
  want to add and adds a random "sleep delay" *before* each call. without this
  delay, the queue gets messed up more easily and videos get lost i.e. they
  don't get added to the queue.

- playing a video from a specific starting time (`t` parameter in urls) works
  only for the *first* video and only if you are *not* using the `-a` (add)
  option.

- `ytcast` doesn't appear in `Settings > Linked devices` menu. it used to show
  up there and there was a button to "unlink all devices" which caused the
  `screenId` to change, but a YouTube update "broke" this feature.

workarounds
-----------

- some devices don't support the DIAL protocol ([notably Chromecast][60]) so
  they can't be discovered by `ytcast`. the YouTube on TV app has a ["link with
  code" functionality][61] which can be used as workaround to pair `ytcast` with
  these devices. the pairing code can be found in `Settings > Link with TV code`
  and then you can use the `-pair` option to do the pairing:

      $ ytcast -pair 123456789101
      8a59f138 unknown         "YouTube on TV"

  once paired you can cast videos in the usual way:

      $ ytcast -d 8a59 https://www.youtube.com/watch?v=t0Q2otsqC4I

  when using this method, `ytcast` and the target device do *not* need to be on
  the same network, but it adds many manual steps i.e. the YouTube on TV app
  must be already open because `ytcast` won't be able to start it nor to
  Wake-On-Lan the TV and it won't automatically "re-pair" when the `screenId`
  changes (I don't know how often that happens).

- playlist urls don't work with `ytcast`, I haven't found a reliable way to pass
  playlist ids to the api. `youtube-dl` comes to the rescue (see also [other
  tools][62]) since it can extract all video urls of YouTube playlists:

      $ youtube-dl -j --flat-playlist https://www.youtube.com/playlist?list=PLrOv9FMX8xJHqMvSGB_9G9nZZ_4IgteYf | jq -r '.url' | ytcast -p

  you can of course filter the pipeline as you like and this makes it so
  flexible that I actually don't feel the need for `ytcast` to support playlist
  urls: less is more!

- this might sound obvious, but if you are tired of typing the device name (even
  a substring of it) every time you want to cast something or if you have
  multiple devices with the same name, you can define shell aliases and benefit
  from shell tab-completion feature:

      $ alias ytcbed="ytcast -d 'LG 32'"
      $ ytcbed https://www.youtube.com/watch?v=dQw4w9WgXcQ

  (see your shell documentation to make aliases persistent, usually you have to
  add them in the shell's `rc` file).

[60]: https://en.wikipedia.org/wiki/Chromecast#Device_discovery_protocols
[61]: https://support.google.com/youtube/answer/3230451
[62]: #other-tools

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

other tools
-----------

as I said earlier, my computing environment is very command-line centric and I'd
like to showcase the other tools I use to enjoy a "no frills" YouTube experience
from the terminal!

- [`youtube-dl`][40] (actually [`yt-dlp`][41] these days) doesn't need
  introduction, it's an awesome tool and it's well integrated with [`mpv`][42]
  so I can watch videos with my favorite player without having my laptop fan
  spin like an airplane engine thanks to this `mpv` config:

      ytdl-format=bestvideo[height<=?1080][vcodec!=?vp9]+bestaudio/best

- [`ytsearch`][43] is my clone of the initial version of [`ytfzf`][44]. it
  allows to search and select video urls from the command-line using the
  wonderful [`fzf`][45] (fun fact: it's implemented basically as a single big
  pipe ahah). you have already seen it in action in `ytcast` examples, but it
  works great with `mpv` too:

      $ ytsearch matrix 4 | xargs mpv
      $ ytsearch 9 symphony | xargs mpv --no-video

- [`ytxrss`][46] allows to extract the rss feed url of a YouTube channel
  starting from a video or channel url. I use rss feeds ([`newsboat`][47]) to
  keep up-to-date with *things* and I'm really glad YouTube still supports them
  for channel uploads. if I'm interested in a channel's future uploads, what I
  usually do is:

      $ ytxrss https://www.youtube.com/user/Computerphile >> ~/.newsboat/urls

[40]: https://github.com/ytdl-org/youtube-dl
[41]: https://github.com/yt-dlp/yt-dlp
[42]: https://github.com/mpv-player/mpv
[43]: https://github.com/MarcoLucidi01/bin/blob/master/ytsearch
[44]: https://github.com/pystardust/ytfzf
[45]: https://github.com/junegunn/fzf
[46]: https://github.com/MarcoLucidi01/bin/blob/master/ytxrss
[47]: https://github.com/newsboat/newsboat

---

see [license file][50] for copyright and license details.

[50]: license
