package fs

import (
	"context"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/operations"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"regexp"
	"strings"
)

// List files
func list(ctx context.Context, path string) ([]model.Obj, error) {
	meta := ctx.Value("meta").(*model.Meta)
	user := ctx.Value("user").(*model.User)
	account, actualPath, err := operations.GetAccountAndActualPath(path)
	virtualFiles := operations.GetAccountVirtualFilesByPath(path)
	if err != nil {
		if len(virtualFiles) != 0 {
			return virtualFiles, nil
		}
		return nil, errors.WithMessage(err, "failed get account")
	}
	objs, err := operations.List(ctx, account, actualPath)
	if err != nil {
		log.Errorf("%+v", err)
		if len(virtualFiles) != 0 {
			return virtualFiles, nil
		}
		return nil, errors.WithMessage(err, "failed get objs")
	}
	for _, accountFile := range virtualFiles {
		if !containsByName(objs, accountFile) {
			objs = append(objs, accountFile)
		}
	}
	if whetherHide(user, meta, path) {
		objs = hide(objs, meta)
	}
	// sort objs
	if account.Config().LocalSort {
		model.SortFiles(objs, account.GetAccount().OrderBy, account.GetAccount().OrderDirection)
	}
	model.ExtractFolder(objs, account.GetAccount().ExtractFolder)
	return objs, nil
}

func whetherHide(user *model.User, meta *model.Meta, path string) bool {
	// if is admin, don't hide
	if user.CanSeeHides() {
		return false
	}
	// if meta is nil, don't hide
	if meta == nil {
		return false
	}
	// if meta.Hide is empty, don't hide
	if meta.Hide == "" {
		return false
	}
	// if meta doesn't apply to sub_folder, don't hide
	if !utils.PathEqual(meta.Path, path) && !meta.HSub {
		return false
	}
	// if is guest, hide
	return true
}

func hide(objs []model.Obj, meta *model.Meta) []model.Obj {
	var res []model.Obj
	deleted := make([]bool, len(objs))
	rs := strings.Split(meta.Hide, "\n")
	for _, r := range rs {
		re, _ := regexp.Compile(r)
		for i, obj := range objs {
			if deleted[i] {
				continue
			}
			if re.MatchString(obj.GetName()) {
				deleted[i] = true
			}
		}
	}
	for i, obj := range objs {
		if !deleted[i] {
			res = append(res, obj)
		}
	}
	return res
}
