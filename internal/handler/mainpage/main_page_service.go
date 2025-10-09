package mainpage

type MainPageService interface {
	GetMainPage() (string, error)
}
