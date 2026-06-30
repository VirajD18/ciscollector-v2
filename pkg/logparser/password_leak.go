package logparser

import (
	"context"

	"github.com/VirajD18/ciscollector-v2/pkg/config"
	"github.com/VirajD18/ciscollector-v2/pkg/parselog"
)

type PasswordLeakHelper struct {
	*parselog.PasswordLeakParser
}

func NewPasswordLeakHelper() *PasswordLeakHelper {
	return &PasswordLeakHelper{}
}

func (i *PasswordLeakHelper) Init(ctx context.Context, logParserCnf *config.LogParser) error {

	i.PasswordLeakParser = parselog.NewPasswordLeakParser(logParserCnf)
	return nil
}

func (i *PasswordLeakHelper) GetResult(ctx context.Context) []parselog.LeakedPasswordResponse {
	return i.PasswordLeakParser.GetLeakedPasswords()
}
