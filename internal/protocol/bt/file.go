// modify from github.com/anacrolix/torrent/storage/file.go

package bt

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/anacrolix/torrent/storage"

	"github.com/anacrolix/missinggo/v2"

	"github.com/anacrolix/torrent/common"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/segments"
)

// File-based storage for torrents, that isn't yet bound to a particular torrent.
type fileClientImpl struct {
	opts newFileClientOpts
}

type newFileClientOpts struct {
	// The base directory for all downloads.
	ClientBaseDir     string
	HandleFileTorrent func(infoHash metainfo.Hash, ft *fileTorrentImpl)
	FilePathMaker     storage.FilePathMaker
	PieceCompletion   storage.PieceCompletion
}

func newFileOpts(opts newFileClientOpts) storage.ClientImplCloser {
	if opts.FilePathMaker == nil {
		opts.FilePathMaker = func(opts storage.FilePathMakerOpts) string {
			var parts []string
			if opts.Info.Length == 0 || opts.File.Path == nil {
				parts = append(parts, opts.Info.Name)
			}
			return filepath.Join(append(parts, opts.File.Path...)...)
		}
	}
	if opts.PieceCompletion == nil {
		ret, err := storage.NewDefaultPieceCompletionForDir(opts.ClientBaseDir)
		if err != nil {
			ret = storage.NewMapPieceCompletion()
		}
		opts.PieceCompletion = ret
	}
	return fileClientImpl{opts}
}

func (fs fileClientImpl) Close() error {
	return fs.opts.PieceCompletion.Close()
}

func (fs fileClientImpl) OpenTorrent(ctx context.Context, info *metainfo.Info, infoHash metainfo.Hash) (_ storage.TorrentImpl, err error) {
	upvertedFiles := info.UpvertedFiles()
	files := make([]file, 0, len(upvertedFiles))
	for _, fileInfo := range upvertedFiles {
		filePath := filepath.Join("", fs.opts.FilePathMaker(storage.FilePathMakerOpts{
			Info: info,
			File: &fileInfo,
		}))
		f := file{
			rawPath: filePath,
			path:    filePath,
			length:  fileInfo.Length,
		}
		if f.length == 0 {
			err = CreateNativeZeroLengthFile(f.path)
			if err != nil {
				err = fmt.Errorf("creating zero length file: %w", err)
				return
			}
		}
		files = append(files, f)
	}
	t := &fileTorrentImpl{
		files,
		segments.NewIndex(common.LengthIterFromUpvertedFiles(upvertedFiles)),
		infoHash,
		fs.opts.PieceCompletion,
	}
	fs.opts.HandleFileTorrent(infoHash, t)
	return storage.TorrentImpl{
		Piece: t.Piece,
		Close: t.Close,
	}, nil
}

type file struct {
	// The safe, OS-local file path.
	rawPath string
	path    string
	length  int64
}

type fileTorrentImpl struct {
	files          []file
	segmentLocater segments.Index
	infoHash       metainfo.Hash
	completion     storage.PieceCompletion
}

func (fts *fileTorrentImpl) Piece(p metainfo.Piece) storage.PieceImpl {
	// Create a view onto the file-based torrent storage.
	_io := fileTorrentImplIO{fts}
	// Return the appropriate segments of this.
	return &filePieceImpl{
		fts,
		p,
		missinggo.NewSectionWriter(_io, p.Offset(), p.Length()),
		io.NewSectionReader(_io, p.Offset(), p.Length()),
	}
}

func (fts *fileTorrentImpl) Close() error {
	return nil
}

func (fts *fileTorrentImpl) setTorrentDir(dir string) {
	for i, f := range fts.files {
		fts.files[i].path = filepath.Join(dir, f.rawPath)
	}
}

// A helper to create zero-length files which won't appear for file-orientated storage since no
// writes will ever occur to them (no torrent data is associated with a zero-length file). The
// caller should make sure the file name provided is safe/sanitized.
func CreateNativeZeroLengthFile(name string) error {
	os.MkdirAll(filepath.Dir(name), 0o777)
	var f io.Closer
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	return f.Close()
}

// Exposes file-based storage of a torrent, as one big ReadWriterAt.
type fileTorrentImplIO struct {
	fts *fileTorrentImpl
}

// Returns EOF on short or missing file.
func (fst *fileTorrentImplIO) readFileAt(file file, b []byte, off int64) (n int, err error) {
	f, err := os.Open(file.path)
	if os.IsNotExist(err) {
		// File missing is treated the same as a short file.
		err = io.EOF
		return
	}
	if err != nil {
		return
	}
	defer f.Close()
	// Limit the read to within the expected bounds of this file.
	if int64(len(b)) > file.length-off {
		b = b[:file.length-off]
	}
	for off < file.length && len(b) != 0 {
		n1, err1 := f.ReadAt(b, off)
		b = b[n1:]
		n += n1
		off += int64(n1)
		if n1 == 0 {
			err = err1
			break
		}
	}
	return
}

// Only returns EOF at the end of the torrent. Premature EOF is ErrUnexpectedEOF.
func (fst fileTorrentImplIO) ReadAt(b []byte, off int64) (n int, err error) {
	fst.fts.segmentLocater.Locate(segments.Extent{off, int64(len(b))}, func(i int, e segments.Extent) bool {
		n1, err1 := fst.readFileAt(fst.fts.files[i], b[:e.Length], e.Start)
		n += n1
		b = b[n1:]
		err = err1
		return err == nil // && int64(n1) == e.Length
	})
	if len(b) != 0 && err == nil {
		err = io.EOF
	}
	return
}

func (fst fileTorrentImplIO) WriteAt(p []byte, off int64) (n int, err error) {
	// log.Printf("write at %v: %v bytes", off, len(p))
	fst.fts.segmentLocater.Locate(segments.Extent{off, int64(len(p))}, func(i int, e segments.Extent) bool {
		name := fst.fts.files[i].path
		os.MkdirAll(filepath.Dir(name), 0o777)
		var f *os.File
		f, err = os.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0o666)
		if err != nil {
			return false
		}
		var n1 int
		n1, err = f.WriteAt(p[:e.Length], e.Start)
		// log.Printf("%v %v wrote %v: %v", i, e, n1, err)
		closeErr := f.Close()
		n += n1
		p = p[n1:]
		if err == nil {
			err = closeErr
		}
		if err == nil && int64(n1) != e.Length {
			err = io.ErrShortWrite
		}
		return err == nil
	})
	return
}
