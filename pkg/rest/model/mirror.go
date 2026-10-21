package model

type MirrorCandidatesReq struct {
	URL string `json:"url"`
}

type MirrorCandidatesResp struct {
	URLs []string `json:"urls"`
}
