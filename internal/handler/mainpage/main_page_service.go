package mainpage

import "context"

type MainPageService interface {
	GetMainPage(ctx context.Context) (string, error)
}
