package bt

import (
	"context"
	"errors"
	"os"
	"sync"

	g "github.com/anacrolix/generics"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
)

// deferredFileStorage lets metadata resolve without opening the user's files.
// It binds once. After binding, pieces are supplied directly by file storage.
type deferredFileStorage struct {
	mu          sync.RWMutex
	ctx         context.Context
	info        *metainfo.Info
	hash        metainfo.Hash
	knownHashes []bool
	owner       storage.ClientImplCloser
	torrent     storage.TorrentImpl
	closed      bool
}

func (s *deferredFileStorage) OpenTorrent(ctx context.Context, info *metainfo.Info, hash metainfo.Hash) (storage.TorrentImpl, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.info != nil {
		return storage.TorrentImpl{}, errors.New("torrent storage already opened or closed")
	}
	s.ctx, s.info, s.hash = ctx, info, hash
	s.knownHashes = make([]bool, info.NumPieces())
	return storage.TorrentImpl{
		PieceWithHash: func(p metainfo.Piece, hash g.Option[[]byte]) storage.PieceImpl {
			s.mu.Lock()
			defer s.mu.Unlock()
			if s.closed {
				return metadataPiece{err: os.ErrClosed}
			}
			if s.owner == nil {
				s.knownHashes[p.Index()] = hash.Ok
				// A missing v2 hash must remain unknown until its arrival changes
				// completion and updates the library's piece request priorities.
				return metadataPiece{known: hash.Ok}
			}
			if s.torrent.PieceWithHash != nil {
				return s.torrent.PieceWithHash(p, hash)
			}
			return s.torrent.Piece(p)
		},
		Close: s.Close,
	}, nil
}

func (s *deferredFileStorage) bind(opts storage.NewFileClientOpts) ([]bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, os.ErrClosed
	}
	if s.info == nil {
		return nil, errors.New("torrent metadata is not ready")
	}
	if s.owner != nil {
		return nil, nil
	}
	if opts.FilePathMaker == nil {
		opts.FilePathMaker = func(o storage.FilePathMakerOpts) string { _, name := torrentFileLayout(o.Info, *o.File); return name }
	}
	owner := storage.NewFileOpts(opts)
	t, err := owner.OpenTorrent(s.ctx, s.info, s.hash)
	if err != nil {
		owner.Close()
		return nil, err
	}
	s.owner, s.torrent = owner, t
	return append([]bool(nil), s.knownHashes...), nil
}

func (s *deferredFileStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.owner == nil {
		return nil
	}
	var err error
	if s.torrent.Close != nil {
		err = s.torrent.Close()
	}
	return errors.Join(err, s.owner.Close())
}

type metadataPiece struct {
	known bool
	err   error
}

func (p metadataPiece) Completion() storage.Completion {
	return storage.Completion{Ok: p.known, Err: p.err}
}
func (p metadataPiece) ReadAt([]byte, int64) (int, error) {
	return 0, errors.New("download storage is not initialized")
}
func (p metadataPiece) WriteAt([]byte, int64) (int, error) {
	return 0, errors.New("download storage is not initialized")
}
func (p metadataPiece) MarkComplete() error { return errors.New("download storage is not initialized") }
func (p metadataPiece) MarkNotComplete() error {
	return errors.New("download storage is not initialized")
}
