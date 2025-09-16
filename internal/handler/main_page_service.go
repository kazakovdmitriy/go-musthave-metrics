package handler

type MainPageService interface {
	GetMainPage() (string, error)
}
