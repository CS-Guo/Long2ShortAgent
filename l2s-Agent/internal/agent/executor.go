package agent

type ShortenerClient struct {
	BaseURL string
}

func NewShortenerClient(baseURL string) *ShortenerClient {
	return &ShortenerClient{
		BaseURL: baseURL,
	}
}

func (c *ShortenerClient) CreateShort(longUrl string) (string, error) {
	payload := map[string]string{"longUrl": longUrl}

}
