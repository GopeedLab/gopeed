package bt

import (
	"fmt"
	"github.com/monkeyWie/gopeed-core/internal/controller"
	"github.com/monkeyWie/gopeed-core/internal/fetcher"
	"github.com/monkeyWie/gopeed-core/pkg/base"
	"testing"
)

func TestFetcher_Resolve_Magnet(t *testing.T) {
	fetcher := buildFetcher()
	res, err := fetcher.Resolve(&base.Request{
		URL: "magnet:?xt=urn:btih:A213E40628B27B58E70F0DADAC5350CA05403AC4&dn=Microsoft%20Office%20PRO%20Plus%202021%20Retail%20Version%202108%20Build%2014326.2&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.openbittorrent.com%3A6969%2Fannounce&tr=udp%3A%2F%2F9.rarbg.to%3A2710%2Fannounce&tr=udp%3A%2F%2F9.rarbg.me%3A2780%2Fannounce&tr=udp%3A%2F%2F9.rarbg.to%3A2730%2Fannounce&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337&tr=http%3A%2F%2Fp4p.arenabg.com%3A1337%2Fannounce&tr=udp%3A%2F%2Ftracker.torrent.eu.org%3A451%2Fannounce&tr=udp%3A%2F%2Ftracker.tiny-vps.com%3A6969%2Fannounce&tr=udp%3A%2F%2Fopen.stealth.si%3A80%2Fannounce",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(res)
}

func TestFetcher_Resolve_Torrent(t *testing.T) {
	fetcher := buildFetcher()
	res, err := fetcher.Resolve(&base.Request{
		URL: "d:/test/test.torrent",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(res)
}

func TestFetcher_Create(t *testing.T) {
	fetcher := buildFetcher()
	res, err := fetcher.Resolve(&base.Request{
		URL: "magnet:?xt=urn:btih:A213E40628B27B58E70F0DADAC5350CA05403AC4&dn=Microsoft%20Office%20PRO%20Plus%202021%20Retail%20Version%202108%20Build%2014326.2&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.openbittorrent.com%3A6969%2Fannounce&tr=udp%3A%2F%2F9.rarbg.to%3A2710%2Fannounce&tr=udp%3A%2F%2F9.rarbg.me%3A2780%2Fannounce&tr=udp%3A%2F%2F9.rarbg.to%3A2730%2Fannounce&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337&tr=http%3A%2F%2Fp4p.arenabg.com%3A1337%2Fannounce&tr=udp%3A%2F%2Ftracker.torrent.eu.org%3A451%2Fannounce&tr=udp%3A%2F%2Ftracker.tiny-vps.com%3A6969%2Fannounce&tr=udp%3A%2F%2Fopen.stealth.si%3A80%2Fannounce",
	})
	if err != nil {
		panic(err)
	}
	if err := fetcher.Create(res, &base.Options{}); err != nil {
		panic(err)
	}
	if err := fetcher.Start(); err != nil {
		panic(err)
	}
	if err := fetcher.Wait(); err != nil {
		panic(err)
	}
}

func buildFetcher() fetcher.Fetcher {
	fetcher := new(FetcherBuilder).Build()
	fetcher.Setup(controller.NewController())
	return fetcher
}
