package forgekshadow

const DefaultMaxReports = 50

type Config struct {
	Enabled    bool
	MaxReports int
}

func (c Config) normalized() Config {
	if c.MaxReports <= 0 {
		c.MaxReports = DefaultMaxReports
	}
	return c
}
