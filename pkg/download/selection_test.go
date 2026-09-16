package download

import (
	"reflect"
	"testing"

	"github.com/GopeedLab/gopeed/internal/fetcher"
	"github.com/GopeedLab/gopeed/pkg/base"
)

func TestCreateResolvedFileSelection(t *testing.T) {
	for _, tc := range []struct {
		name      string
		selection []int
		submit    bool
		want      []int
		size      int64
	}{
		{"partial", []int{2, 0}, true, []int{2, 0}, 40},
		{"default all", nil, true, []int{0, 1, 2}, 60},
		{"legacy retains resolve options", nil, false, []int{1}, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager := &generationTestManager{}
			d := NewDownloader(&DownloaderConfig{FetchManagers: []fetcher.FetcherManager{manager}})
			if err := d.Setup(); err != nil {
				t.Fatal(err)
			}
			defer d.Clear()
			d.cfg.MaxRunning = 0
			original := &base.Options{Path: t.TempDir(), SelectFiles: []int{1}}
			rr, err := d.Resolve(&base.Request{URL: "generation://selection"}, original)
			if err != nil {
				t.Fatal(err)
			}
			d.fetcherCache[rr.ID].Meta().Res = &base.Resource{Name: "bundle", Files: []*base.FileInfo{
				{Name: "a", Size: 10}, {Name: "b", Size: 20}, {Name: "c", Size: 30},
			}}
			var id string
			if tc.submit {
				id, err = d.CreateWithOptions(rr.ID, &base.Options{Path: original.Path, SelectFiles: tc.selection})
			} else {
				id, err = d.Create(rr.ID)
			}
			if err != nil {
				t.Fatal(err)
			}
			task := d.GetTask(id)
			if !reflect.DeepEqual(task.Meta.Opts.SelectFiles, tc.want) {
				t.Fatalf("selection = %v, want %v", task.Meta.Opts.SelectFiles, tc.want)
			}
			if task.Meta.Res.Size != tc.size {
				t.Fatalf("queued size = %d, want %d", task.Meta.Res.Size, tc.size)
			}
			var saved Task
			if _, err := d.storage.Get(bucketTask, id, &saved); err != nil {
				t.Fatal(err)
			}
			if saved.Meta.Res.Size != tc.size || !reflect.DeepEqual(saved.Meta.Opts.SelectFiles, tc.want) {
				t.Fatalf("persisted metadata = %+v", saved.Meta)
			}
		})
	}
}
