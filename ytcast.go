// See license file for copyright and license details.

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
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

	launchTimeout       = 1 * time.Minute
	launchCheckInterval = 3 * time.Second

	fallbackIdFormat = "0405.0000.2006010215" // poor man's UUID.
)

var (
	progVersion = "vX.Y.Z-dev" // set with -ldflags at build time

	errNoDevFound      = errors.New("no device found")
	errNoDevLastUsed   = errors.New("no device last used")
	errNoDevMatch      = errors.New("no device matches")
	errMoreDevMatch    = errors.New("more than one device matches")
	errNoDevSelected   = errors.New("no device selected")
	errNoLaunch        = errors.New("unable to launch app and get screenId")
	errNoVideo         = errors.New("no video to play")
	errUnknownAppState = errors.New("unknown app state")
	errInvalidCode     = errors.New("invalid pairing code")

	flagAdd        = flag.Bool("a", false, "add video(s) to queue, don't change what's currently playing")
	flagClearCache = flag.Bool("c", false, "clear cache")
	flagDevName    = flag.String("d", "", "select device by substring of name, hostname (ip) or unique service name")
	flagLastUsed   = flag.Bool("p", false, "select last used device")
	flagList       = flag.Bool("l", false, "list cached devices")
	flagPairCode   = flag.String("pair", "", "manual pair using TV code, skip device discovery")
	flagSearch     = flag.Bool("s", false, "search (discover) devices on the network and update cache")
	flagTimeout    = flag.Duration("t", dial.MSearchMinTimeout, fmt.Sprintf("search timeout (max %s)", dial.MSearchMaxTimeout))
	flagVerbose    = flag.Bool("verbose", false, "enable verbose logging")
	flagVersion    = flag.Bool("v", false, "print program version")
)

// cast contains a dial.Device and the youtube.Remote connected to that Device.
// Device will be nil if Remote was manually paired using a TV code.
// It's stored in the cache.
type cast struct {
	Device   *dial.Device
	Remote   *youtube.Remote
	LastUsed bool // true if Device is the last successfully used Device.
	cached   bool // true if Device was fetched from the cache and not just discovered/updated.
}

func main() {
	flag.StringVar(flagDevName, "n", "", "deprecated, same as -d")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [-a|-c|-d|-l|-p|-s|-t|-v|-pair|-verbose] [video...]\n\n", progName)
		fmt.Fprintf(flag.CommandLine.Output(), "cast YouTube videos to your smart TV.\n\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\n%s %s\n%s\n", progName, progVersion, progRepo)
	}
	flag.Parse()

	if *flagVersion {
		fmt.Printf("%s %s\n", progName, progVersion)
		return
	}
	log.SetFlags(log.Ltime | log.Lshortfile)
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
	cacheFilePath := filepath.Join(mkCacheDir(), cacheFileName)
	cache := make(map[string]*cast)
	if !*flagClearCache {
		cache = loadCache(cacheFilePath)
	}
	defer saveCache(cacheFilePath, cache)

	if *flagPairCode != "" {
		return manualPair(cache, *flagPairCode)
	}
	if len(cache) == 0 || *flagSearch {
		if err := discoverDevices(cache, *flagTimeout); err != nil {
			return err
		}
	}

	var selected *cast
	var err error
	switch {
	case *flagDevName != "":
		if selected, err = matchOneDevice(cache, *flagDevName); err == nil {
			break
		}
		if !errors.Is(err, errNoDevMatch) {
			return err
		}
		if err = discoverDevices(cache, *flagTimeout); err != nil {
			return err
		}
		if len(cache) == 0 {
			return errNoDevFound
		}
		if selected, err = matchOneDevice(cache, *flagDevName); err != nil {
			return err
		}

	case *flagLastUsed:
		if selected = findLastUsedDevice(cache); selected == nil {
			return errNoDevLastUsed
		}

	case len(cache) == 0:
		// this check is done here and NOT immediately after the first
		// discoverDevices() to give a chance to rediscover in -d case.
		return errNoDevFound

	case *flagList, *flagSearch:
		listDevices(cache)
		return nil

	default:
		listDevices(cache)
		return errNoDevSelected
	}

	videos := flag.Args()
	if len(videos) == 0 || (len(videos) == 1 && videos[0] == "-") {
		if videos, err = readVideosFromStdin(); err != nil {
			return err
		}
		if len(videos) == 0 {
			return errNoVideo
		}
	}

	screenId := ""
	if selected.wasManuallyPaired() {
		// try to reuse the screenId since we can't know if it changed.
		screenId = selected.Remote.ScreenId
	} else {
		if !selected.Device.Ping() {
			log.Printf("%q is not awake, trying waking it up...", selected.name())
			if err := selected.Device.TryWakeup(); err != nil {
				return fmt.Errorf("%q: TryWakeup: %w", selected.name(), err)
			}
		}
		if screenId, err = launchYouTubeApp(selected.Device); err != nil {
			return err
		}
	}
	for _, entry := range cache {
		entry.LastUsed = entry == selected
	}

	if needsToConnect(selected.Remote, screenId) {
		log.Printf("connecting to %q via YouTube Lounge", selected.name())
		remote, err := youtube.Connect(screenId, getConnectName())
		if err != nil {
			return fmt.Errorf("Connect: %w", err)
		}
		if selected.wasManuallyPaired() {
			// these fields must be maintained because they are not
			// returned by Connect(), but only by ConnectWithCode().
			remote.DeviceId = selected.Remote.DeviceId
			remote.ScreenName = selected.Remote.ScreenName
		}
		selected.Remote = remote
	}
	if *flagAdd {
		log.Printf("requesting YouTube Lounge to add %v to %q's playing queue", videos, selected.name())
		if err := selected.Remote.Add(videos); err != nil {
			return fmt.Errorf("Add: %w", err)
		}
		return nil
	}
	log.Printf("requesting YouTube Lounge to play %v on %q", videos, selected.name())
	if err := selected.Remote.Play(videos); err != nil {
		return fmt.Errorf("Play: %w", err)
	}
	return nil
}

func mkCacheDir() string {
	cacheDir := os.Getenv(xdgCache)
	if cacheDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Println(err)
			return "." // current directory
		}
		cacheDir = filepath.Join(homeDir, fallbackCacheDir)
	}
	cacheDir = filepath.Join(cacheDir, progName)
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
		cache[entry.uuid()] = entry
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

func manualPair(cache map[string]*cast, code string) error {
	if code = strings.TrimSpace(code); code == "" {
		return errInvalidCode
	}
	log.Println("connecting to device via YouTube Lounge and pairing code")
	remote, err := youtube.ConnectWithCode(code, getConnectName())
	if err != nil {
		return fmt.Errorf("ConnectWithCode: %w", err)
	}
	if remote.DeviceId == "" {
		remote.DeviceId = strings.ReplaceAll(time.Now().Format(fallbackIdFormat), ".", "")
	}
	if remote.ScreenName == "" {
		remote.ScreenName = remote.DeviceId
	}
	if entry, ok := cache[remote.DeviceId]; ok {
		entry.Remote = remote
		entry.cached = false
	} else {
		cache[remote.DeviceId] = &cast{Remote: remote}
	}
	fmt.Println(cache[remote.DeviceId])
	return nil
}

func discoverDevices(cache map[string]*cast, timeout time.Duration) error {
	devCh, err := dial.Discover(nil, timeout)
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

func matchOneDevice(cache map[string]*cast, name string) (*cast, error) {
	nameLow := strings.ToLower(strings.TrimSpace(name))
	var matched []*cast
	for _, entry := range cache {
		matches := strings.Contains(strings.ToLower(entry.name()), nameLow) ||
			strings.Contains(strings.ToLower(entry.hostname()), nameLow) ||
			strings.Contains(strings.ToLower(entry.uuid()), nameLow)
		if matches {
			matched = append(matched, entry)
		}
	}
	if len(matched) == 1 {
		return matched[0], nil
	}
	if len(matched) == 0 {
		return nil, fmt.Errorf("%w %q", errNoDevMatch, name)
	}
	var matchedStr strings.Builder
	for _, m := range matched {
		matchedStr.WriteRune('\n')
		matchedStr.WriteString(m.String())
	}
	return nil, fmt.Errorf("%w %q:%s", errMoreDevMatch, name, matchedStr.String())
}

func findLastUsedDevice(cache map[string]*cast) *cast {
	for _, entry := range cache {
		if entry.LastUsed {
			return entry
		}
	}
	return nil
}

func listDevices(cache map[string]*cast) {
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
		return entries[i].name() < entries[j].name()
	})
	for _, entry := range entries {
		fmt.Println(entry)
	}
}

func (c *cast) wasManuallyPaired() bool {
	return c.Device == nil // implies c.Remote != nil
}

func (c *cast) uuid() string {
	if c.wasManuallyPaired() {
		return c.Remote.DeviceId
	}
	return c.Device.UniqueServiceName
}

func (c *cast) name() string {
	if c.wasManuallyPaired() {
		return c.Remote.ScreenName
	}
	return c.Device.FriendlyName
}

func (c *cast) hostname() string {
	if c.wasManuallyPaired() {
		return "unknown"
	}
	return c.Device.Hostname()
}

func (c *cast) String() string {
	var info []string
	if c.cached {
		info = append(info, "cached")
	}
	if c.LastUsed {
		info = append(info, "lastused")
	}
	return fmt.Sprintf("%.8s %-15s %-30q %s",
		strings.TrimPrefix(c.uuid(), "uuid:"), c.hostname(), c.name(), strings.Join(info, " "))
}

func launchYouTubeApp(dev *dial.Device) (string, error) {
	for start := time.Now(); time.Since(start) < launchTimeout; time.Sleep(launchCheckInterval) {
		app, err := dev.GetAppInfo(youtube.DialAppName, youtube.Origin)
		if err != nil {
			return "", fmt.Errorf("%q: GetAppInfo: %q: %w", dev.FriendlyName, youtube.DialAppName, err)
		}

		log.Printf("%q is %s on %q", youtube.DialAppName, app.State, dev.FriendlyName)
		switch app.State {
		case "running":
			screenId, err := youtube.ExtractScreenId(app.Additional.Data)
			if err != nil {
				return "", err
			}
			if screenId != "" {
				return screenId, nil
			}
			log.Println("screenId not available")

		case "stopped", "hidden":
			log.Printf("launching %q on %q", youtube.DialAppName, dev.FriendlyName)
			if _, err := dev.Launch(youtube.DialAppName, youtube.Origin, ""); err != nil {
				return "", fmt.Errorf("%q: Launch: %q: %w", dev.FriendlyName, youtube.DialAppName, err)
			}

		default:
			return "", fmt.Errorf("%q: %q: %q: %w", dev.FriendlyName, youtube.DialAppName, app.State, errUnknownAppState)
		}
	}
	return "", fmt.Errorf("%q: %q: %w", dev.FriendlyName, youtube.DialAppName, errNoLaunch)
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

func needsToConnect(remote *youtube.Remote, screenId string) bool {
	switch {
	case remote == nil:
		return true
	case remote.ScreenId != screenId:
		log.Println("screenId changed")
		return true
	case remote.Expired():
		log.Println("LoungeToken expired, trying refreshing it")
		if err := remote.RefreshToken(); err != nil {
			log.Printf("RefreshToken: %s", err)
			return true
		}
	}
	return false
}

func getConnectName() string {
	u, err := user.Current()
	if err != nil {
		log.Println(err)
		return progName
	}
	h, err := os.Hostname()
	if err != nil {
		log.Println(err)
		return progName
	}
	return fmt.Sprintf("%s@%s", u.Username, h)
}
