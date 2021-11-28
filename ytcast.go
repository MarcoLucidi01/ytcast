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
	"strconv"
	"strings"
	"time"

	"github.com/MarcoLucidi01/ytcast/dial"
	"github.com/MarcoLucidi01/ytcast/youtube"
)

const (
	progName            = "ytcast"
	cacheFileName       = progName + ".json"
	launchTimeout       = 1 * time.Minute
	launchCheckInterval = 2 * time.Second
)

type cacheEntry struct {
	Device   *dial.Device
	Remote   *youtube.Remote
	LastUsed bool
	cached   bool
}

var (
	errCanceled        = errors.New("canceled")
	errUnknownAppState = errors.New("unknown app state")
	errNoLaunch        = fmt.Errorf("unable to launch %s app and get screenId", youtube.DialAppName)

	flagVerbose = flag.Bool("verbose", false, "enable verbose logging")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [-v] [videoId...]\n", progName)
		flag.PrintDefaults()
	}
	flag.Parse()

	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Lshortfile)
	if !*flagVerbose {
		log.SetOutput(ioutil.Discard)
	}

	if err := run(); err != nil {
		log.Println(err)
		fmt.Fprintf(os.Stderr, "%s: %s\n", progName, err)
		os.Exit(1)
	}
}

func run() error {
	cacheFilePath := path.Join(mkCacheDir(), cacheFileName)
	cache := loadCache(cacheFilePath)
	// cache slice will likely change, cannot defer saveCache() directly
	defer func() { saveCache(cacheFilePath, cache) }()

	entry, err := selectDevice(&cache)
	if err != nil {
		return err
	}
	if err := entry.Device.WakeupFunc(); err != nil {
		return err
	}
	screenId, err := launchYouTubeApp(entry)
	if err != nil {
		return err
	}
	for _, e := range cache {
		e.LastUsed = e == entry
	}
	if flag.NArg() > 0 {
		return playVideos(entry, screenId, flag.Args())
	}
	return nil
}

func mkCacheDir() string {
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Println(err)
			return "." // current directory
		}
		cacheDir = path.Join(homeDir, ".cache")
	}
	cacheDir = path.Join(cacheDir, progName)
	log.Printf("mkdir -p %s", cacheDir)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Println(err)
		return "."
	}
	return cacheDir
}

func loadCache(fpath string) []*cacheEntry {
	log.Printf("loading cache %s", fpath)
	data, err := ioutil.ReadFile(fpath)
	if err != nil {
		log.Println(err)
		return nil
	}
	var cache []*cacheEntry
	if err = json.Unmarshal(data, &cache); err != nil {
		log.Println(err)
		return nil
	}
	for _, entry := range cache {
		entry.cached = true
	}
	return cache
}

func saveCache(fpath string, cache []*cacheEntry) {
	log.Printf("saving cache %s", fpath)
	data, err := json.Marshal(cache)
	if err != nil {
		log.Printf("marshal cache: %s", err)
		return
	}
	if err := ioutil.WriteFile(fpath, data, 0600); err != nil {
		log.Println(err)
	}
}

func selectDevice(cache *[]*cacheEntry) (*cacheEntry, error) {
	refresh := len(*cache) == 0
	timeout := 2 * time.Second
	for {
		if refresh {
			if err := discoverDevices(cache, timeout); err != nil {
				return nil, err
			}
			timeout += 1 * time.Second
		}
		refresh = true // on next iteration

		showDevices(*cache)
		if len(*cache) == 0 {
			if err := askRefreshOrCancel(); err != nil {
				return nil, err
			}
			continue
		}
		dev, err := askWhichDevice(*cache)
		if err != nil {
			return nil, err
		}
		if dev != nil {
			return dev, nil
		}
	}
}

func discoverDevices(cache *[]*cacheEntry, timeout time.Duration) error {
	devCh, err := dial.Discover(timeout)
	if err != nil {
		return err
	}

	// make a map to easily find devices for cache update
	cacheMap := make(map[string]*cacheEntry)
	for _, entry := range *cache {
		cacheMap[entry.Device.UniqueServiceName] = entry
	}

	for dev := range devCh {
		if entry, ok := cacheMap[dev.UniqueServiceName]; ok {
			entry.Device = dev
			entry.cached = false
		} else {
			cacheMap[dev.UniqueServiceName] = &cacheEntry{Device: dev}
		}
	}

	// update original slice
	*cache = (*cache)[:0]
	for _, entry := range cacheMap {
		*cache = append(*cache, entry)
	}

	return nil
}

func showDevices(cache []*cacheEntry) {
	if len(cache) == 0 {
		fmt.Println("no device found!")
		return
	}
	sort.Slice(cache, func(i, j int) bool {
		switch {
		case cache[i].LastUsed:
			return true
		case cache[j].LastUsed:
			return false
		case !cache[i].cached && cache[j].cached:
			return true
		case cache[i].cached && !cache[j].cached:
			return false
		}
		return cache[i].Device.FriendlyName < cache[j].Device.FriendlyName
	})
	for i, entry := range cache {
		var info []string
		host := entry.Device.ApplicationUrl
		if u, err := url.Parse(entry.Device.ApplicationUrl); err == nil {
			host = u.Hostname()
		}
		info = append(info, host)
		if entry.cached {
			info = append(info, "cached")
		}
		if entry.LastUsed {
			info = append(info, "lastused")
		}
		fmt.Printf("[%d] %-30s (%s)\n", i, entry.Device.FriendlyName, strings.Join(info, ", "))
	}
}

func ask(question string) (string, error) {
	s := bufio.NewScanner(os.Stdin)
	fmt.Print(question)
	s.Scan()
	return strings.TrimSpace(s.Text()), s.Err()
}

func askRefreshOrCancel() error {
	for {
		input, err := ask("(R refresh, C cancel): ")
		if err != nil {
			return err
		}
		switch input {
		case "R", "r":
			return nil
		case "C", "c":
			return errCanceled
		}
	}
}

func askWhichDevice(cache []*cacheEntry) (*cacheEntry, error) {
	for {
		input, err := ask("which device? (default 0, R refresh, C cancel): ")
		if err != nil {
			return nil, err
		}
		switch input {
		case "":
			return cache[0], nil
		case "R", "r":
			return nil, nil
		case "C", "c":
			return nil, errCanceled
		default:
			i, err := strconv.Atoi(input)
			if err == nil && i >= 0 && i < len(cache) {
				return cache[i], nil
			}
		}
	}
}

func launchYouTubeApp(entry *cacheEntry) (string, error) {
	for start := time.Now(); time.Since(start) < launchTimeout; time.Sleep(launchCheckInterval) {
		app, err := entry.Device.GetAppInfo(youtube.DialAppName, youtube.Origin)
		if err != nil {
			return "", err
		}
		switch app.State {
		case "running":
			log.Printf("%s app is running on %q", youtube.DialAppName, entry.Device.UniqueServiceName)
			screenId, err := extractScreenId(app.Additional.Data)
			if err != nil {
				return "", err
			}
			if screenId != "" {
				return screenId, nil
			}
			log.Println("screenId still not available")
		case "stopped", "hidden":
			if _, err := entry.Device.Launch(youtube.DialAppName, youtube.Origin, ""); err != nil {
				return "", err
			}
		default:
			return "", fmt.Errorf("%s app: %s: %w", youtube.DialAppName, app.State, errUnknownAppState)
		}
	}
	return "", errNoLaunch
}

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

func playVideos(entry *cacheEntry, screenId string, videoIds []string) error {
	remote := entry.Remote
	if remote == nil || remote.ScreenId != screenId || remote.Expired() {
		if remote != nil {
			log.Println("unable to reuse cached YouTube Lounge session")
		}
		var err error
		if remote, err = youtube.Connect(screenId, progName); err != nil {
			return err
		}
		entry.Remote = remote
	}
	for i, v := range videoIds {
		videoIds[i] = extractVideoId(v)
	}
	return remote.Play(videoIds...)
}

func extractVideoId(v string) string {
	v = strings.TrimSpace(v)
	u, err := url.Parse(v)
	if err != nil {
		return v
	}
	vid := u.Query().Get("v")
	if vid != "" {
		return vid
	}
	if vid = path.Base(u.Path); vid != "" && vid != "." && vid != "/" {
		return vid
	}
	return v
}
