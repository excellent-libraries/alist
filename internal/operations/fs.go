package operations

import (
	"context"
	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/errs"
	log "github.com/sirupsen/logrus"
	"os"
	stdpath "path"
	"strings"
	"time"

	"github.com/Xhofe/go-cache"
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/singleflight"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/pkg/errors"
)

// In order to facilitate adding some other things before and after file operations

var filesCache = cache.NewMemCache(cache.WithShards[[]model.Obj](64))
var filesG singleflight.Group[[]model.Obj]

func ClearCache(account driver.Driver, path string) {
	key := stdpath.Join(account.GetAccount().VirtualPath, path)
	filesCache.Del(key)
}

// List files in storage, not contains virtual file
func List(ctx context.Context, account driver.Driver, path string, refresh ...bool) ([]model.Obj, error) {
	path = utils.StandardizePath(path)
	log.Debugf("operations.List %s", path)
	dir, err := Get(ctx, account, path)
	if err != nil {
		return nil, errors.WithMessage(err, "failed get dir")
	}
	if !dir.IsDir() {
		return nil, errors.WithStack(errs.NotFolder)
	}
	if account.Config().NoCache {
		return account.List(ctx, dir)
	}
	key := stdpath.Join(account.GetAccount().VirtualPath, path)
	if len(refresh) == 0 || !refresh[0] {
		if files, ok := filesCache.Get(key); ok {
			return files, nil
		}
	}
	files, err, _ := filesG.Do(key, func() ([]model.Obj, error) {
		files, err := account.List(ctx, dir)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to list files")
		}
		// TODO: maybe can get duration from account's config
		filesCache.Set(key, files, cache.WithEx[[]model.Obj](time.Minute*time.Duration(conf.Conf.CaCheExpiration)))
		return files, nil
	})
	return files, err
}

func isRoot(path, rootFolderPath string) bool {
	if utils.PathEqual(path, rootFolderPath) {
		return true
	}
	rootFolderPath = strings.TrimSuffix(rootFolderPath, "/")
	rootFolderPath = strings.TrimPrefix(rootFolderPath, "\\")
	// relative path, this shouldn't happen, because root folder path is absolute
	if utils.PathEqual(path, "/") && rootFolderPath == "." {
		return true
	}
	return false
}

// Get object from list of files
func Get(ctx context.Context, account driver.Driver, path string) (model.Obj, error) {
	path = utils.StandardizePath(path)
	log.Debugf("operations.Get %s", path)
	if g, ok := account.(driver.Getter); ok {
		return g.Get(ctx, path)
	}
	// is root folder
	if r, ok := account.GetAddition().(driver.IRootFolderId); ok && utils.PathEqual(path, "/") {
		return model.Object{
			ID:       r.GetRootFolderId(),
			Name:     "root",
			Size:     0,
			Modified: account.GetAccount().Modified,
			IsFolder: true,
		}, nil
	}
	if r, ok := account.GetAddition().(driver.IRootFolderPath); ok && isRoot(path, r.GetRootFolderPath()) {
		return model.Object{
			ID:       r.GetRootFolderPath(),
			Name:     "root",
			Size:     0,
			Modified: account.GetAccount().Modified,
			IsFolder: true,
		}, nil
	}
	// not root folder
	dir, name := stdpath.Split(path)
	files, err := List(ctx, account, dir)
	if err != nil {
		return nil, errors.WithMessage(err, "failed get parent list")
	}
	for _, f := range files {
		if f.GetName() == name {
			// use path as id, why don't set id in List function?
			// because files maybe cache, set id here can reduce memory usage
			if f.GetID() == "" {
				if s, ok := f.(model.SetID); ok {
					s.SetID(path)
				}
			}
			return f, nil
		}
	}
	return nil, errors.WithStack(errs.ObjectNotFound)
}

var linkCache = cache.NewMemCache(cache.WithShards[*model.Link](16))
var linkG singleflight.Group[*model.Link]

// Link get link, if is an url. should have an expiry time
func Link(ctx context.Context, account driver.Driver, path string, args model.LinkArgs) (*model.Link, model.Obj, error) {
	file, err := Get(ctx, account, path)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "failed to get file")
	}
	if file.IsDir() {
		return nil, nil, errors.WithStack(errs.NotFile)
	}
	key := stdpath.Join(account.GetAccount().VirtualPath, path)
	if link, ok := linkCache.Get(key); ok {
		return link, file, nil
	}
	fn := func() (*model.Link, error) {
		link, err := account.Link(ctx, file, args)
		if err != nil {
			return nil, errors.WithMessage(err, "failed get link")
		}
		if link.Expiration != nil {
			linkCache.Set(key, link, cache.WithEx[*model.Link](*link.Expiration))
		}
		return link, nil
	}
	link, err, _ := linkG.Do(key, fn)
	return link, file, err
}

func MakeDir(ctx context.Context, account driver.Driver, path string) error {
	// check if dir exists
	f, err := Get(ctx, account, path)
	if err != nil {
		if errs.IsObjectNotFound(err) {
			parentPath, dirName := stdpath.Split(path)
			err = MakeDir(ctx, account, parentPath)
			if err != nil {
				return errors.WithMessagef(err, "failed to make parent dir [%s]", parentPath)
			}
			parentDir, err := Get(ctx, account, parentPath)
			// this should not happen
			if err != nil {
				return errors.WithMessagef(err, "failed to get parent dir [%s]", parentPath)
			}
			return account.MakeDir(ctx, parentDir, dirName)
		} else {
			return errors.WithMessage(err, "failed to check if dir exists")
		}
	} else {
		// dir exists
		if f.IsDir() {
			return nil
		} else {
			// dir to make is a file
			return errors.New("file exists")
		}
	}
}

func Move(ctx context.Context, account driver.Driver, srcPath, dstDirPath string) error {
	srcObj, err := Get(ctx, account, srcPath)
	if err != nil {
		return errors.WithMessage(err, "failed to get src object")
	}
	dstDir, err := Get(ctx, account, dstDirPath)
	if err != nil {
		return errors.WithMessage(err, "failed to get dst dir")
	}
	return account.Move(ctx, srcObj, dstDir)
}

func Rename(ctx context.Context, account driver.Driver, srcPath, dstName string) error {
	srcObj, err := Get(ctx, account, srcPath)
	if err != nil {
		return errors.WithMessage(err, "failed to get src object")
	}
	return account.Rename(ctx, srcObj, dstName)
}

// Copy Just copy file[s] in an account
func Copy(ctx context.Context, account driver.Driver, srcPath, dstDirPath string) error {
	srcObj, err := Get(ctx, account, srcPath)
	if err != nil {
		return errors.WithMessage(err, "failed to get src object")
	}
	dstDir, err := Get(ctx, account, dstDirPath)
	return account.Copy(ctx, srcObj, dstDir)
}

func Remove(ctx context.Context, account driver.Driver, path string) error {
	obj, err := Get(ctx, account, path)
	if err != nil {
		// if object not found, it's ok
		if errs.IsObjectNotFound(err) {
			return nil
		}
		return errors.WithMessage(err, "failed to get object")
	}
	return account.Remove(ctx, obj)
}

func Put(ctx context.Context, account driver.Driver, dstDirPath string, file model.FileStreamer, up driver.UpdateProgress) error {
	defer func() {
		if f, ok := file.GetReadCloser().(*os.File); ok {
			err := os.RemoveAll(f.Name())
			if err != nil {
				log.Errorf("failed to remove file [%s]", f.Name())
			}
		}
	}()
	defer func() {
		if err := file.Close(); err != nil {
			log.Errorf("failed to close file streamer, %v", err)
		}
	}()
	err := MakeDir(ctx, account, dstDirPath)
	if err != nil {
		return errors.WithMessagef(err, "failed to make dir [%s]", dstDirPath)
	}
	parentDir, err := Get(ctx, account, dstDirPath)
	// this should not happen
	if err != nil {
		return errors.WithMessagef(err, "failed to get dir [%s]", dstDirPath)
	}
	// if up is nil, set a default to prevent panic
	if up == nil {
		up = func(p int) {}
	}
	err = account.Put(ctx, parentDir, file, up)
	log.Debugf("put file [%s] done", file.GetName())
	if err == nil {
		// clear cache
		key := stdpath.Join(account.GetAccount().VirtualPath, dstDirPath)
		filesCache.Del(key)
	}
	return err
}
