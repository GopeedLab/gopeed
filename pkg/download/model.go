package download

import (
	"encoding/json"
	"github.com/GopeedLab/gopeed/internal/controller"
	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/internal/protocol/bt"
	"github.com/GopeedLab/gopeed/internal/protocol/http"
	"github.com/GopeedLab/gopeed/pkg/base"
	"github.com/GopeedLab/gopeed/pkg/util"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"sync"
	"time"
)

type ResolveResult struct {
	ID  string         `json:"id"`
	Res *base.Resource `json:"res"`
}

type Task struct {
	ID        string               `json:"id"`
	Protocol  string               `json:"protocol"`
	Meta      *fetcher.FetcherMeta `json:"meta"`
	Status    base.Status          `json:"status"`
	Uploading bool                 `json:"uploading"`
	Progress  *Progress            `json:"progress"`
	CreatedAt time.Time            `json:"createdAt"`
	UpdatedAt time.Time            `json:"updatedAt"`

	fetcherManager fetcher.FetcherManager
	fetcher        fetcher.Fetcher
	timer          *util.Timer
	statusLock     *sync.Mutex
	lock           *sync.Mutex
	speedArr       []int64
	uploadSpeedArr []int64
}

func NewTask() *Task {
	id, err := gonanoid.New()
	if err != nil {
		panic(err)
	}
	return &Task{
		ID:        id,
		Status:    base.DownloadStatusReady,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Name returns the display name of the task.
func (t *Task) Name() string {
	// Custom name first
	if t.Meta.Opts.Name != "" {
		return t.Meta.Opts.Name
	}

	// Task is not resolved, parse the name from the URL
	if t.Meta.Res == nil {
		fallbackName := "unknown"
		if t.fetcherManager == nil {
			return fallbackName
		}
		parseName := t.fetcherManager.ParseName(t.Meta.Req.URL)
		if parseName == "" {
			return fallbackName
		}
		return parseName
	}

	// Task is a folder
	if t.Meta.Res.Name != "" {
		return t.Meta.Res.Name
	}

	// Get the name of the first file
	return t.Meta.Res.Files[0].Name
}

func (t *Task) MarshalJSON() ([]byte, error) {
	type rawTaskType Task
	jsonTask := struct {
		rawTaskType
		Name string `json:"name"`
	}{
		rawTaskType(*t),
		t.Name(),
	}
	return json.Marshal(jsonTask)
}

func (t *Task) updateStatus(status base.Status) {
	t.UpdatedAt = time.Now()
	t.Status = status
}

func (t *Task) clone() *Task {
	return util.DeepClone(t)
}

func (t *Task) updateSpeed(downloaded int64, usedTime float64) int64 {
	return calcSpeed(&t.speedArr, downloaded, usedTime)
}

func (t *Task) updateUploadSpeed(downloaded int64, usedTime float64) int64 {
	return calcSpeed(&t.uploadSpeedArr, downloaded, usedTime)
}

func calcSpeed(speedArr *[]int64, downloaded int64, usedTime float64) int64 {
	*speedArr = append(*speedArr, downloaded)
	// Record last 5 seconds of download speed to calculate the average speed
	if len(*speedArr) > int(5.0/usedTime) {
		*speedArr = (*speedArr)[1:]
	}

	var total int64
	for _, v := range *speedArr {
		total += v
	}

	return int64(float64(total) / float64(len(*speedArr)) / usedTime)
}

type TaskFilter struct {
	IDs         []string
	Statuses    []base.Status
	NotStatuses []base.Status
}

func (f *TaskFilter) IsEmpty() bool {
	return len(f.IDs) == 0 && len(f.Statuses) == 0 && len(f.NotStatuses) == 0
}

type DownloaderConfig struct {
	Controller    *controller.Controller
	FetchManagers []fetcher.FetcherManager

	RefreshInterval   int // RefreshInterval time duration to refresh task progress(ms)
	Storage           Storage
	StorageDir        string
	WhiteDownloadDirs []string

	ProductionMode bool

	*base.DownloaderStoreConfig
}

func (cfg *DownloaderConfig) Init() *DownloaderConfig {
	if cfg.Controller == nil {
		cfg.Controller = controller.NewController()
	}
	if len(cfg.FetchManagers) == 0 {
		cfg.FetchManagers = []fetcher.FetcherManager{
			new(http.FetcherManager),
			new(bt.FetcherManager),
		}
	}
	if cfg.RefreshInterval == 0 {
		cfg.RefreshInterval = 350
	}
	if cfg.Storage == nil {
		cfg.Storage = NewMemStorage()
	}
	return cfg
}
