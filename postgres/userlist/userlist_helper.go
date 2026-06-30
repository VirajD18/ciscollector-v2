package userlist

import (
	"context"
	"database/sql"

	"github.com/VirajD18/ciscollector-v2/model"
	"github.com/VirajD18/ciscollector-v2/pkg/utils"
)

type UserlistHelper struct {
	Title string
	Note  string
	Query string
	List  bool
}

func (u *UserlistHelper) Process(db *sql.DB, ctx context.Context) *model.UserlistResult {
	result := &model.UserlistResult{
		Title: u.Title,
	}

	if !u.List {
		data, err := utils.GetTableResponse(db, u.Query)
		if err != nil {
			result.Data = model.ManualCheckTableDescriptionAndList{Description: u.Note, List: []string{err.Error()}}
			return result
		}

		result.Data = model.ManualCheckTableDescriptionAndList{
			Description: u.Note,
			Table:       data,
		}
		return result
	}

	list := []string{}
	rows, err := db.QueryContext(ctx, u.Query)
	if err != nil {
		result.Data = model.ManualCheckTableDescriptionAndList{Description: u.Note, List: []string{err.Error()}}
		return result
	}

	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			result.Data = model.ManualCheckTableDescriptionAndList{Description: u.Note, List: []string{err.Error()}}
			return result
		}
		list = append(list, v)
	}

	result.Data = model.ManualCheckTableDescriptionAndList{
		Description: u.Note,
		List:        list,
	}

	return result
}
