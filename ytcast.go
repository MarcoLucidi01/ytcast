// See license file for copyright and license details.

package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/MarcoLucidi01/ytcast/dial"
	"github.com/MarcoLucidi01/ytcast/youtube"
)

const (
	progName = "ytcast"
	progRepo = "https://github.com/MarcoLucidi01/ytcast"

	xdgCache         = "XDG_CACHE_HOME"
	fallbackCacheDir = ".cache" // used if xdgCache is not set, stored in $HOME
	cacheFileName    = progName + ".json"

	searchTimeout       = 3 * time.Second
	launchTimeout       = 1 * time.Minute
	launchCheckInterval = 2 * time.Second
)

// cast contains a dial.Device and the youtube.Remote connected to that Device.
// It's stored in the cache.
type cast struct {
	Device   *dial.Device
	Remote   *youtube.Remote
	LastUsed bool // true if Device is the last successfully used Device.
	cached   bool // true if Device was fetched from the cache and not just discovered/updated.
}

var (
	progVersion = "develop" // set with -ldflags at build time

	errNoDevFound      = errors.New("no device found")
	errNoDevLastUsed   = errors.New("no device last used")
	errNoDevMatch      = errors.New("no device matches")
	errNoDevSelected   = errors.New("no device selected")
	errNoLaunch        = errors.New("unable to launch app and get screenId")
	errNoVideo         = errors.New("no video to play")
	errUnknownAppState = errors.New("unknown app state")

	flagLastUsed = flag.Bool("l", false, "select last used device")
	flagName     = flag.String("n", "", "select device by substring of name, hostname (ip) or unique service name")
	flagSearch   = flag.Bool("s", false, "search (discover) devices on the network and update cache")
	flagTimeout  = flag.Duration("t", searchTimeout, fmt.Sprintf("search timeout (min %s max %s)", dial.MsMinTimeout, dial.MsMaxTimeout))
	flagVerbose  = flag.Bool("verbose", false, "enable verbose logging")
	flagVersion  = flag.Bool("v", false, "print program version")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [-l|-n|-s|-t|-v|-verbose] [video...]\n\n", progName)
		fmt.Fprintf(flag.CommandLine.Output(), "cast YouTube videos to your smart TV.\n\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\n%s\n", progRepo)
	}
	flag.Parse()

	if *flagVersion {
		fmt.Fprintf(os.Stderr, "%s %s\n", progName, progVersion)
		return
	}
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Lshortfile)
	if !*flagVerbose {
		log.SetOutput(ioutil.Discard)
	}
	log.Printf("%s %s\n", progName, progVersion)

	if err := run(); err != nil {
		log.Println(err)
		fmt.Fprintf(os.Stderr, "%s: %s\n", progName, err)
		os.Exit(1)
	}
}

func run() error {
	cacheFilePath := path.Join(mkCacheDir(), cacheFileName)
	cache := loadCache(cacheFilePath)
	defer saveCache(cacheFilePath, cache)

	if len(cache) == 0 || *flagSearch {
		if err := discoverDevices(cache, *flagTimeout); err != nil {
			return err
		}
		if len(cache) == 0 {
			return errNoDevFound
		}
		if *flagSearch {
			showDevices(cache)
			return nil
		}
	}

	var selected *cast
	switch {
	case *flagName != "":
		if selected = matchDevice(cache, *flagName); selected == nil {
			return fmt.Errorf("%w %q", errNoDevMatch, *flagName)
		}
	case *flagLastUsed:
		if selected = findLastUsedDevice(cache); selected == nil {
			return errNoDevLastUsed
		}
	default:
		showDevices(cache)
		return errNoDevSelected
	}

	videos := flag.Args()
	if len(videos) == 0 || (len(videos) == 1 && videos[0] == "-") {
		var err error
		if videos, err = readVideosFromStdin(); err != nil {
			return err
		}
		if len(videos) == 0 {
			return errNoVideo
		}
	}

	if !selected.Device.Ping() {
		log.Printf("%q is not awake, trying waking it up...", selected.Device.FriendlyName)
		if err := selected.Device.TryWakeup(); err != nil {
			return fmt.Errorf("%q: TryWakeup: %w", selected.Device.FriendlyName, err)
		}
	}
	screenId, err := launchYouTubeApp(selected)
	if err != nil {
		return err
	}
	for _, entry := range cache {
		entry.LastUsed = entry == selected
	}
	return playVideos(selected, screenId, videos)
}

func mkCacheDir() string {
	cacheDir := os.Getenv(xdgCache)
	if cacheDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Println(err)
			return "." // current directory
		}
		cacheDir = path.Join(homeDir, fallbackCacheDir)
	}
	cacheDir = path.Join(cacheDir, progName)
	log.Printf("mkdir -p %s", cacheDir)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Println(err)
		return "."
	}
	return cacheDir
}

func loadCache(fpath string) map[string]*cast {
	log.Printf("loading cache %s", fpath)
	cache := make(map[string]*cast)
	data, err := ioutil.ReadFile(fpath)
	if err != nil {
		log.Println(err)
		return cache
	}
	var cacheValues []*cast
	if err = json.Unmarshal(data, &cacheValues); err != nil {
		log.Printf("unmarshal cache: %s", err)
		return cache
	}
	for _, entry := range cacheValues {
		entry.cached = true
		cache[entry.Device.UniqueServiceName] = entry
	}
	return cache
}

func saveCache(fpath string, cache map[string]*cast) {
	log.Printf("saving cache %s", fpath)
	var cacheValues []*cast
	for _, entry := range cache {
		cacheValues = append(cacheValues, entry)
	}
	data, err := json.Marshal(cacheValues)
	if err != nil {
		log.Printf("marshal cache: %s", err)
		return
	}
	if err := ioutil.WriteFile(fpath, data, 0600); err != nil {
		log.Println(err)
	}
}

func discoverDevices(cache map[string]*cast, timeout time.Duration) error {
	devCh, err := dial.Discover(timeout)
	if err != nil {
		return fmt.Errorf("Discover: %w", err)
	}
	for dev := range devCh {
		if entry, ok := cache[dev.UniqueServiceName]; ok {
			entry.Device = dev
			entry.cached = false
		} else {
			cache[dev.UniqueServiceName] = &cast{Device: dev}
		}
	}
	return nil
}

func matchDevice(cache map[string]*cast, name string) *cast {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, entry := range cache {
		if strings.Contains(strings.ToLower(entry.Device.FriendlyName), name) {
			return entry
		}
	}
	for _, entry := range cache {
		// TODO dial.Device should have a hostname field already parsed
		// from ApplicationUrl for convenience
		host := entry.Device.ApplicationUrl
		if u, err := url.Parse(entry.Device.ApplicationUrl); err == nil {
			host = u.Hostname()
		}
		if strings.Contains(strings.ToLower(host), name) {
			return entry
		}
	}
	for _, entry := range cache {
		if strings.Contains(strings.ToLower(entry.Device.UniqueServiceName), name) {
			return entry
		}
	}
	return nil
}

func findLastUsedDevice(cache map[string]*cast) *cast {
	for _, entry := range cache {
		if entry.LastUsed {
			return entry
		}
	}
	return nil
}

func showDevices(cache map[string]*cast) {
	var entries []*cast
	for _, entry := range cache {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		switch {
		case !entries[i].cached && entries[j].cached:
			return true
		case entries[i].cached && !entries[j].cached:
			return false
		case entries[i].LastUsed:
			return true
		case entries[j].LastUsed:
			return false
		}
		return entries[i].Device.FriendlyName < entries[j].Device.FriendlyName
	})
	for _, entry := range entries {
		fmt.Println(entry)
	}
}

func (c *cast) String() string {
	var info []string
	host := c.Device.ApplicationUrl
	if u, err := url.Parse(c.Device.ApplicationUrl); err == nil {
		host = u.Hostname()
	}
	info = append(info, host)
	if c.cached {
		info = append(info, "cached")
	}
	if c.LastUsed {
		info = append(info, "lastused")
	}
	return fmt.Sprintf("%-30q %s", c.Device.FriendlyName, strings.Join(info, " "))
}

func launchYouTubeApp(selected *cast) (string, error) {
	appName := youtube.DialAppName
	devName := selected.Device.FriendlyName
	for start := time.Now(); time.Since(start) < launchTimeout; time.Sleep(launchCheckInterval) {
		app, err := selected.Device.GetAppInfo(appName, youtube.Origin)
		if err != nil {
			return "", fmt.Errorf("%q: GetAppInfo: %q: %w", devName, appName, err)
		}

		log.Printf("%q is %s on %q", appName, app.State, devName)
		switch app.State {
		case "running":
			screenId, err := extractScreenId(app.Additional.Data)
			if err != nil {
				return "", err
			}
			if screenId != "" {
				return screenId, nil
			}
			log.Println("screenId not available")

		case "stopped", "hidden":
			log.Printf("launching %q on %q", appName, devName)
			if _, err := selected.Device.Launch(appName, youtube.Origin, ""); err != nil {
				return "", fmt.Errorf("%q: Launch: %q: %w", devName, appName, err)
			}

		default:
			return "", fmt.Errorf("%q: %q: %q: %w", devName, appName, app.State, errUnknownAppState)
		}
	}
	return "", fmt.Errorf("%q: %q: %w", devName, appName, errNoLaunch)
}

// TODO move to youtube package?
func extractScreenId(data string) (string, error) {
	// TODO dial.AppInfo.Additional.Data it's not wrapped in a root element,
	// I add a dummy root here but I think data should already be wrapped in
	// a root element.
	data = fmt.Sprintf("<dummy>%s</dummy>", data)
	var v struct {
		ScreenId string `xml:"screenId"`
	}
	if err := xml.Unmarshal([]byte(data), &v); err != nil {
		return "", err
	}
	return strings.TrimSpace(v.ScreenId), nil
}

func readVideosFromStdin() ([]string, error) {
	log.Println("reading videos from stdin")
	scanner := bufio.NewScanner(os.Stdin)
	var videos []string
	for scanner.Scan() {
		if v := strings.TrimSpace(scanner.Text()); v != "" {
			videos = append(videos, v)
		}
	}
	return videos, scanner.Err()
}

func playVideos(selected *cast, screenId string, videos []string) error {
	doConnect := false
	switch {
	case selected.Remote == nil:
		doConnect = true
	case selected.Remote.ScreenId != screenId:
		log.Println("screenId changed")
		doConnect = true
	case selected.Remote.Expired():
		// TODO not sure if after refreshing the token, the Remote can
		// actually be used again, I have to test.
		log.Println("LoungeToken expired, trying refreshing it")
		if err := selected.Remote.RefreshToken(); err != nil {
			log.Printf("RefreshToken: %s", err)
			doConnect = true
		}
	}
	if doConnect {
		log.Printf("connecting to %q via YouTube Lounge", selected.Device.FriendlyName)
		// TODO use "$USER@$HOSTNAME progName" instead of just progName?
		remote, err := youtube.Connect(screenId, progName)
		if err != nil {
			return fmt.Errorf("Connect: %w", err)
		}
		selected.Remote = remote
	}
	for i, v := range videos {
		videos[i] = extractVideoId(v)
	}
	log.Printf("requesting YouTube Lounge to play %v on %q", videos, selected.Device.FriendlyName)
	if err := selected.Remote.Play(videos...); err != nil {
		return fmt.Errorf("Play: %w", err)
	}
	return nil
}

// TODO move to youtube package?
func extractVideoId(video string) string {
	video = strings.TrimSpace(video)
	u, err := url.Parse(video)
	if err != nil {
		return video
	}
	vid := u.Query().Get("v")
	if vid != "" {
		return vid
	}
	if vid = path.Base(u.Path); vid != "" && vid != "." && vid != "/" {
		return vid
	}
	return video
}
